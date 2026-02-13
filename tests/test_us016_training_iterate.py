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

        if "Improve the following agent based on evaluation feedback" in message_content:
            if "conservative approach, prioritize safety" in message_content:
                response_text = "evolved lineage A"
            elif "balanced approach, equal priority to effectiveness and safety" in message_content:
                response_text = "evolved lineage B"
            elif "creative approach, prioritize novel solutions" in message_content:
                response_text = "evolved lineage C"
            elif "aggressive approach, prioritize speed and efficiency" in message_content:
                response_text = "evolved lineage D"
            else:
                response_text = "evolved lineage unknown"
        else:
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


def _lineage_by_name(session_doc: dict, name: str) -> dict:
    for lineage in session_doc["lineages"].values():
        if lineage["name"] == name:
            return lineage
    raise AssertionError(f"lineage {name} missing")


def test_us016_training_iterate_regenerates_only_unlocked_lineages(tmp_path: Path):
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

        iterate = run(
            (
                f"{binary} training iterate {session_id} "
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert iterate.returncode == 0, iterate.stderr
        assert "Regenerated 3 lineages: B, C, D. Locked: A." in iterate.stdout

        state_doc = json.loads((tmp_path / ".ludus-magnus" / "state.json").read_text())
        session_doc = state_doc["sessions"][session_id]

        lineage_a = _lineage_by_name(session_doc, "A")
        lineage_b = _lineage_by_name(session_doc, "B")
        lineage_c = _lineage_by_name(session_doc, "C")
        lineage_d = _lineage_by_name(session_doc, "D")

        assert len(lineage_a["agents"]) == 1
        assert len(lineage_b["agents"]) == 2
        assert len(lineage_c["agents"]) == 2
        assert len(lineage_d["agents"]) == 2

        assert lineage_b["agents"][1]["version"] == 2
        assert lineage_c["agents"][1]["version"] == 2
        assert lineage_d["agents"][1]["version"] == 2
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
