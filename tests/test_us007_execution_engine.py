import json
import os
import re
import subprocess
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path


def run(cmd: str, cwd: Path, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    go_path_prefix = 'export PATH="/usr/local/go/bin:$PATH"; '
    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)
    return subprocess.run(
        ["bash", "-lc", f"{go_path_prefix}{cmd}"],
        cwd=cwd,
        capture_output=True,
        text=True,
        env=merged_env,
    )


class _MockOpenAIHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/chat/completions":
            self.send_response(404)
            self.end_headers()
            return

        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))))
        messages = body.get("messages", [])
        system = ""
        user = ""
        for msg in messages:
            if msg.get("role") == "system":
                system = msg.get("content", "")
            if msg.get("role") == "user":
                user = msg.get("content", "")

        if system:
            content = f"ECHO: {user}"
        else:
            content = "You are an echo agent."

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        payload = {
            "choices": [{"message": {"content": content}}],
            "usage": {"prompt_tokens": 10, "completion_tokens": 6, "total_tokens": 16},
        }
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def log_message(self, format, *args):
        return


def _start_server() -> tuple[HTTPServer, threading.Thread]:
    server = HTTPServer(("127.0.0.1", 0), _MockOpenAIHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def test_us007_run_execution_engine(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"
    server, thread = _start_server()

    try:
        engine_tests = run("go test ./internal/engine -v", cwd=repo_root)
        assert engine_tests.returncode == 0, engine_tests.stderr
        assert "PASS" in engine_tests.stdout

        build = run("go build -o ludus-magnus", cwd=repo_root)
        assert build.returncode == 0, build.stderr

        init = run(
            (
                f'{binary} quickstart init --need "echo agent" '
                f'--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}'
            ),
            cwd=tmp_path,
        )
        assert init.returncode == 0, init.stderr
        session_match = re.search(r"session_id=(ses_[a-f0-9]{8})", init.stdout)
        assert session_match
        session_id = session_match.group(1)

        api_run = run(
            (
                f'{binary} run {session_id} --input "hello" '
                f'--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}'
            ),
            cwd=tmp_path,
        )
        assert api_run.returncode == 0, api_run.stderr
        assert re.search(r"artifact_id=art_[a-f0-9]{8}", api_run.stdout)

        state_file = tmp_path / ".ludus-magnus" / "state.json"
        state_doc = json.loads(state_file.read_text())
        session = state_doc["sessions"][session_id]
        lineage_key = next(iter(session["lineages"].keys()))
        artifact = session["lineages"][lineage_key]["artifacts"][0]
        assert artifact["output"] == "ECHO: hello"
        assert artifact["execution_metadata"]["mode"] == "api"
        assert artifact["execution_metadata"]["provider"] == "openai-compatible"

        # Convert the session to training mode and reuse the lineage as A.
        session["mode"] = "training"
        session["lineages"][lineage_key]["name"] = "A"
        state_file.write_text(json.dumps(state_doc, indent=2) + "\n")

        training_run = run(
            (
                f'{binary} run {session_id} --lineage A --input "hello training" '
                f'--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}'
            ),
            cwd=tmp_path,
        )
        assert training_run.returncode == 0, training_run.stderr

        # CLI mode with a fake codex executor.
        bin_dir = tmp_path / "bin"
        bin_dir.mkdir()
        codex = bin_dir / "codex"
        codex.write_text("#!/usr/bin/env bash\ncat >/dev/null\necho codex-mock-response\n")
        codex.chmod(0o755)

        cli_run = run(
            f'{binary} run {session_id} --lineage A --input "hello cli" --mode cli --executor codex',
            cwd=tmp_path,
            env={"PATH": f"{bin_dir}:{os.environ['PATH']}"},
        )
        assert cli_run.returncode == 0, cli_run.stderr

        state_doc = json.loads(state_file.read_text())
        artifacts = state_doc["sessions"][session_id]["lineages"][lineage_key]["artifacts"]
        cli_artifact = artifacts[-1]
        assert cli_artifact["output"] == "codex-mock-response"
        assert cli_artifact["execution_metadata"]["mode"] == "cli"
        assert cli_artifact["execution_metadata"]["executor"] == "codex"
        assert "codex" in cli_artifact["execution_metadata"]["executor_command"]
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
