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
            "choices": [{"message": {"content": "unique-id-ok"}}],
            "usage": {"prompt_tokens": 8, "completion_tokens": 4, "total_tokens": 12},
        }
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def log_message(self, format, *args):
        return


def _start_server() -> tuple[HTTPServer, threading.Thread]:
    server = HTTPServer(("127.0.0.1", 0), _MockOpenAIHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def test_us009a_enforces_unique_ids_and_safe_lookup(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"
    server, thread = _start_server()
    try:
        state_tests = run("go test ./internal/state -v", cwd=repo_root)
        assert state_tests.returncode == 0, state_tests.stderr

        build = run("go build -o ludus-magnus", cwd=repo_root)
        assert build.returncode == 0, build.stderr

        init = run(
            (
                f'{binary} quickstart init --need "us009a" '
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert init.returncode == 0, init.stderr
        session_id = re.search(r"session_id=(ses_[a-f0-9]{8})", init.stdout).group(1)

        run_one = run(
            (
                f'{binary} run {session_id} --input "first" '
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert run_one.returncode == 0, run_one.stderr
        artifact_one = re.search(r"artifact_id=(art_[a-f0-9]{8})", run_one.stdout).group(1)

        run_two = run(
            (
                f'{binary} run {session_id} --input "second" '
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert run_two.returncode == 0, run_two.stderr
        artifact_two = re.search(r"artifact_id=(art_[a-f0-9]{8})", run_two.stdout).group(1)
        assert artifact_one != artifact_two

        state_file = tmp_path / ".ludus-magnus" / "state.json"
        state_doc = json.loads(state_file.read_text())
        state_doc["sessions"]["ses_dup1"] = {
            "id": "ses_dup1",
            "mode": "quickstart",
            "need": "dup1",
            "created_at": "2026-02-13T10:30:00Z",
            "status": "active",
            "lineages": {
                "main": {
                    "id": "lin_dup1",
                    "session_id": "ses_dup1",
                    "name": "main",
                    "locked": False,
                    "agents": [],
                    "artifacts": [
                        {"id": "art_collision", "agent_id": "agt_1", "input": "x", "output": "y", "created_at": "2026-02-13T10:31:00Z", "execution_metadata": {"mode": "api", "provider": None, "executor": None, "executor_command": None, "tokens_input": 0, "tokens_output": 0, "duration_ms": 0, "cost_usd": 0, "tool_calls": []}},
                    ],
                    "directives": {"oneshot": [], "sticky": []},
                }
            },
        }
        state_doc["sessions"]["ses_dup2"] = {
            "id": "ses_dup2",
            "mode": "quickstart",
            "need": "dup2",
            "created_at": "2026-02-13T10:32:00Z",
            "status": "active",
            "lineages": {
                "main": {
                    "id": "lin_dup2",
                    "session_id": "ses_dup2",
                    "name": "main",
                    "locked": False,
                    "agents": [],
                    "artifacts": [
                        {"id": "art_collision", "agent_id": "agt_2", "input": "p", "output": "q", "created_at": "2026-02-13T10:33:00Z", "execution_metadata": {"mode": "api", "provider": None, "executor": None, "executor_command": None, "tokens_input": 0, "tokens_output": 0, "duration_ms": 0, "cost_usd": 0, "tool_calls": []}},
                    ],
                    "directives": {"oneshot": [], "sticky": []},
                }
            },
        }
        state_file.write_text(json.dumps(state_doc))

        inspect = run(f"{binary} artifact inspect art_collision", cwd=tmp_path)
        assert inspect.returncode != 0
        assert "not unique" in (inspect.stderr + inspect.stdout)

        evaluate = run(f"{binary} evaluate art_collision --score 7", cwd=tmp_path)
        assert evaluate.returncode != 0
        assert "not unique" in (evaluate.stderr + evaluate.stdout)
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
