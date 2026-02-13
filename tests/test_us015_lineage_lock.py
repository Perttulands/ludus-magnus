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

        response_text = "generic training agent"
        for marker in [
            "conservative approach, prioritize safety",
            "balanced approach, equal priority to effectiveness and safety",
            "creative approach, prioritize novel solutions",
            "aggressive approach, prioritize speed and efficiency",
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


def test_us015_lineage_lock_unlock_and_missing_lineage(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"
    server, thread = _start_server()

    try:
        build = run("go build -o ludus-magnus", cwd=repo_root)
        assert build.returncode == 0, build.stderr

        init = run(
            (
                f'{binary} training init --need "customer care agent" '
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert init.returncode == 0, init.stderr

        session_match = re.search(r"session_id=(ses_[a-f0-9]{8})", init.stdout)
        assert session_match
        session_id = session_match.group(1)

        lock = run(f"{binary} lineage lock {session_id} A", cwd=tmp_path)
        assert lock.returncode == 0, lock.stderr
        assert "Lineage A locked" in lock.stdout

        inspect_locked = run(f"{binary} session inspect {session_id}", cwd=tmp_path)
        assert inspect_locked.returncode == 0, inspect_locked.stderr
        inspect_locked_doc = json.loads(inspect_locked.stdout)

        lineage_a = None
        for lineage in inspect_locked_doc["lineages"].values():
            if lineage["name"] == "A":
                lineage_a = lineage
                break

        assert lineage_a is not None
        assert lineage_a["locked"] is True

        unlock = run(f"{binary} lineage unlock {session_id} A", cwd=tmp_path)
        assert unlock.returncode == 0, unlock.stderr
        assert "Lineage A unlocked" in unlock.stdout

        inspect_unlocked = run(f"{binary} session inspect {session_id}", cwd=tmp_path)
        assert inspect_unlocked.returncode == 0, inspect_unlocked.stderr
        inspect_unlocked_doc = json.loads(inspect_unlocked.stdout)

        lineage_a = None
        for lineage in inspect_unlocked_doc["lineages"].values():
            if lineage["name"] == "A":
                lineage_a = lineage
                break

        assert lineage_a is not None
        assert lineage_a["locked"] is False

        missing = run(f"{binary} lineage lock {session_id} nonexistent", cwd=tmp_path)
        assert missing.returncode != 0
        assert 'lineage "nonexistent" not found' in missing.stderr
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
