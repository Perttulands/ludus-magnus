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
            "choices": [{"message": {"content": "artifact-commands-ok"}}],
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


def test_us011_artifact_list_and_inspect_commands(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"
    server, thread = _start_server()

    try:
        build = run("go build -o ludus-magnus", cwd=repo_root)
        assert build.returncode == 0, build.stderr

        init = run(
            (
                f'{binary} quickstart init --need "artifact commands" '
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert init.returncode == 0, init.stderr
        session_match = re.search(r"session_id=(ses_[a-f0-9]{8})", init.stdout)
        assert session_match
        session_id = session_match.group(1)

        execute = run(
            (
                f'{binary} run {session_id} --input "artifact inspection" '
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert execute.returncode == 0, execute.stderr
        artifact_match = re.search(r"artifact_id=(art_[a-f0-9]{8})", execute.stdout)
        assert artifact_match
        artifact_id = artifact_match.group(1)

        evaluate = run(
            f'{binary} evaluate {artifact_id} --score 8 --comment "solid result"',
            cwd=tmp_path,
        )
        assert evaluate.returncode == 0, evaluate.stderr

        artifact_list = run(f"{binary} artifact list {session_id}", cwd=tmp_path)
        assert artifact_list.returncode == 0, artifact_list.stderr
        assert "ID" in artifact_list.stdout
        assert "Agent Version" in artifact_list.stdout
        assert "Score" in artifact_list.stdout
        assert "Created At" in artifact_list.stdout
        assert artifact_id in artifact_list.stdout
        assert "8" in artifact_list.stdout

        inspect = run(f"{binary} artifact inspect {artifact_id}", cwd=tmp_path)
        assert inspect.returncode == 0, inspect.stderr
        artifact_doc = json.loads(inspect.stdout)
        assert artifact_doc["id"] == artifact_id
        assert artifact_doc["input"] == "artifact inspection"
        assert artifact_doc["output"] == "artifact-commands-ok"
        assert artifact_doc["execution_metadata"]["tokens_input"] > 0
        assert artifact_doc["evaluation"]["score"] == 8
        assert artifact_doc["evaluation"]["comment"] == "solid result"

        missing_session = run(f"{binary} artifact list ses_doesnotexist", cwd=tmp_path)
        assert missing_session.returncode != 0
        assert "session not found" in (missing_session.stderr + missing_session.stdout)
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
