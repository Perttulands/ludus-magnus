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


def test_us014_training_init_creates_four_lineages_with_variants(tmp_path: Path):
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

        for lineage_name in ["A", "B", "C", "D"]:
            assert re.search(rf"lineage_{lineage_name}_id=(lin_[a-f0-9]{{8}})", init.stdout)

        inspect = run(f"{binary} session inspect {session_id}", cwd=tmp_path)
        assert inspect.returncode == 0, inspect.stderr
        inspect_doc = json.loads(inspect.stdout)
        assert len(inspect_doc["lineages"]) == 4

        prompts = []
        for lineage in inspect_doc["lineages"].values():
            assert lineage["locked"] is False
            assert len(lineage["agents"]) == 1
            prompts.append(lineage["agents"][0]["definition"]["system_prompt"])

        assert len(set(prompts)) == 4
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
