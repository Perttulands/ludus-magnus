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

        self.rfile.read(int(self.headers.get("Content-Length", "0")))

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        payload = {
            "choices": [{"message": {"content": "observability-ok"}}],
            "usage": {"prompt_tokens": 15, "completion_tokens": 10, "total_tokens": 25},
        }
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def log_message(self, format, *args):
        return


def _start_server() -> tuple[HTTPServer, threading.Thread]:
    server = HTTPServer(("127.0.0.1", 0), _MockOpenAIHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def test_us008_observability_capture(tmp_path: Path):
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
                f'{binary} quickstart init --need "observe execution" '
                f'--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}'
            ),
            cwd=tmp_path,
        )
        assert init.returncode == 0, init.stderr
        session_match = re.search(r"session_id=(ses_[a-f0-9]{8})", init.stdout)
        assert session_match
        session_id = session_match.group(1)

        execute = run(
            (
                f'{binary} run {session_id} --input "test observability" '
                f'--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}'
            ),
            cwd=tmp_path,
        )
        assert execute.returncode == 0, execute.stderr
        assert re.search(r"artifact_id=art_[a-f0-9]{8}", execute.stdout)

        state_file = tmp_path / ".ludus-magnus" / "state.json"
        state_doc = json.loads(state_file.read_text())
        session = state_doc["sessions"][session_id]
        lineage_key = next(iter(session["lineages"].keys()))
        artifact = session["lineages"][lineage_key]["artifacts"][0]
        metadata = artifact["execution_metadata"]

        assert metadata["tokens_input"] == 15
        assert metadata["tokens_output"] == 10
        assert metadata["duration_ms"] >= 0
        assert metadata["cost_usd"] > 0
        assert isinstance(metadata["tool_calls"], list)
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
