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
            "choices": [{"message": {"content": "evaluation-ready-output"}}],
            "usage": {"prompt_tokens": 9, "completion_tokens": 5, "total_tokens": 14},
        }
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def log_message(self, format, *args):
        return


def _start_server() -> tuple[HTTPServer, threading.Thread]:
    server = HTTPServer(("127.0.0.1", 0), _MockOpenAIHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def _create_artifact(repo_root: Path, tmp_path: Path, binary: Path, port: int) -> tuple[str, str]:
    init = run(
        (
            f'{binary} quickstart init --need "evaluation commands" '
            f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{port}"
        ),
        cwd=tmp_path,
    )
    assert init.returncode == 0, init.stderr
    session_match = re.search(r"session_id=(ses_[a-f0-9]{8})", init.stdout)
    assert session_match
    session_id = session_match.group(1)

    execute = run(
        (
            f'{binary} run {session_id} --input "test artifact" '
            f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{port}"
        ),
        cwd=tmp_path,
    )
    assert execute.returncode == 0, execute.stderr
    artifact_match = re.search(r"artifact_id=(art_[a-f0-9]{8})", execute.stdout)
    assert artifact_match
    return session_id, artifact_match.group(1)


def test_us010_evaluate_command(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"
    server, thread = _start_server()

    try:
        build = run("go build -o ludus-magnus", cwd=repo_root)
        assert build.returncode == 0, build.stderr

        session_id, artifact_id = _create_artifact(repo_root, tmp_path, binary, server.server_port)
        assert session_id

        evaluate = run(
            f'{binary} evaluate {artifact_id} --score 7 --comment "good but needs improvement"',
            cwd=tmp_path,
        )
        assert evaluate.returncode == 0, evaluate.stderr
        assert f"Artifact {artifact_id} evaluated: 7/10" in evaluate.stdout

        state_file = tmp_path / ".ludus-magnus" / "state.json"
        state_doc = json.loads(state_file.read_text())
        session = state_doc["sessions"][session_id]
        lineage_key = next(iter(session["lineages"].keys()))
        artifact = session["lineages"][lineage_key]["artifacts"][0]
        assert artifact["evaluation"]["score"] == 7
        assert artifact["evaluation"]["comment"] == "good but needs improvement"
        assert artifact["evaluation"]["evaluated_at"]

        bad_score = run(f"{binary} evaluate {artifact_id} --score 11", cwd=tmp_path)
        assert bad_score.returncode != 0
        assert "score must be between 1-10" in (bad_score.stderr + bad_score.stdout)

        second_eval = run(f"{binary} evaluate {artifact_id} --score 7", cwd=tmp_path)
        assert second_eval.returncode != 0
        assert "artifact already evaluated" in (second_eval.stderr + second_eval.stdout)
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
