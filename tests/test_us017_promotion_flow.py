import json
import re
import subprocess
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path


def run(cmd: str, cwd: Path) -> subprocess.CompletedProcess[str]:
    go_path_prefix = 'export PATH="/usr/local/go/bin:$PATH"; '
    return subprocess.run(
        ["bash", "-lc", f"{go_path_prefix}{cmd}"],
        cwd=cwd,
        capture_output=True,
        text=True,
    )


class _MockOpenAIHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/chat/completions":
            self.send_response(404)
            self.end_headers()
            return

        body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        request_doc = json.loads(body or b"{}")
        message_content = ""
        messages = request_doc.get("messages") or []
        if messages:
            message_content = messages[-1].get("content", "")

        response_text = "generic generated agent"
        for marker in [
            "conservative approach, prioritize safety",
            "balanced approach, equal priority to effectiveness and safety",
            "creative approach, prioritize novel solutions",
            "aggressive approach, prioritize speed and efficiency",
            "fundamentally different methodology",
        ]:
            if marker in message_content:
                response_text = f"system prompt: {marker}"
                break

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        payload = {
            "choices": [{"message": {"content": response_text}}],
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


def _lineage_by_name(session_doc: dict, name: str) -> dict:
    for lineage in session_doc["lineages"].values():
        if lineage["name"] == name:
            return lineage
    raise AssertionError(f"lineage {name} missing")


def test_us017_promote_quickstart_to_training_variations(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"
    server, thread = _start_server()

    try:
        build = run("go build -o ludus-magnus", cwd=repo_root)
        assert build.returncode == 0, build.stderr

        init = run(
            (
                f'{binary} quickstart init --need "customer care agent" '
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert init.returncode == 0, init.stderr

        session_match = re.search(r"session_id=(ses_[a-f0-9]{8})", init.stdout)
        assert session_match
        session_id = session_match.group(1)

        promote = run(
            (
                f"{binary} promote {session_id} --strategy variations "
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert promote.returncode == 0, promote.stderr
        assert "Session promoted to training mode with 4 lineages" in promote.stdout

        inspect = run(f"{binary} session inspect {session_id}", cwd=tmp_path)
        assert inspect.returncode == 0, inspect.stderr
        inspect_doc = json.loads(inspect.stdout)
        assert inspect_doc["mode"] == "training"
        assert len(inspect_doc["lineages"]) == 4

        lineage_a = _lineage_by_name(inspect_doc, "A")
        assert len(lineage_a["agents"]) == 1
        assert len(lineage_a["agents"][0]["definition"]["system_prompt"]) > 0
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
