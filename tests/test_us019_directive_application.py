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


class _RecordingOpenAIHandler(BaseHTTPRequestHandler):
    evolution_prompts: list[str] = []

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
            _RecordingOpenAIHandler.evolution_prompts.append(message_content)
            response_text = "You are an evolved concise support agent."
        else:
            response_text = "You are a baseline support agent."

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        payload = {
            "choices": [{"message": {"content": response_text}}],
            "usage": {"prompt_tokens": 12, "completion_tokens": 8, "total_tokens": 20},
        }
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def log_message(self, format, *args):
        return


def _start_server() -> tuple[HTTPServer, threading.Thread]:
    server = HTTPServer(("127.0.0.1", 0), _RecordingOpenAIHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def test_us019_directives_are_applied_and_oneshot_is_cleared_after_iterate(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"
    _RecordingOpenAIHandler.evolution_prompts = []
    server, thread = _start_server()

    try:
        build = run("go build -o ludus-magnus", cwd=repo_root)
        assert build.returncode == 0, build.stderr

        init = run(
            (
                f'{binary} quickstart init --need "customer support agent" '
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert init.returncode == 0, init.stderr

        session_match = re.search(r"session_id=(ses_[a-f0-9]{8})", init.stdout)
        assert session_match
        session_id = session_match.group(1)

        sticky = run(
            f'{binary} directive set {session_id} main --text "be concise" --sticky',
            cwd=tmp_path,
        )
        assert sticky.returncode == 0, sticky.stderr

        oneshot = run(
            f'{binary} directive set {session_id} main --text "fix typo" --oneshot',
            cwd=tmp_path,
        )
        assert oneshot.returncode == 0, oneshot.stderr

        iterate = run(
            (
                f"{binary} iterate {session_id} "
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert iterate.returncode == 0, iterate.stderr

        assert _RecordingOpenAIHandler.evolution_prompts
        evolution_prompt = _RecordingOpenAIHandler.evolution_prompts[-1]
        assert "be concise" in evolution_prompt
        assert "fix typo" in evolution_prompt

        state_doc = json.loads((tmp_path / ".ludus-magnus" / "state.json").read_text())
        session_doc = state_doc["sessions"][session_id]

        lineage_doc = None
        for lineage in session_doc["lineages"].values():
            if lineage["name"] == "main":
                lineage_doc = lineage
                break

        assert lineage_doc is not None
        assert [d["text"] for d in lineage_doc["directives"]["sticky"]] == ["be concise"]
        assert lineage_doc["directives"]["oneshot"] == []
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
