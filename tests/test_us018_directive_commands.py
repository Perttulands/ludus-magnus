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

        self.rfile.read(int(self.headers.get("Content-Length", "0")))
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        payload = {
            "choices": [{"message": {"content": "training directive agent"}}],
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


def test_us018_directive_set_clear_and_type_validation(tmp_path: Path):
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

        sticky = run(
            f'{binary} directive set {session_id} A --text "be friendly" --sticky',
            cwd=tmp_path,
        )
        assert sticky.returncode == 0, sticky.stderr
        sticky_id_match = re.search(r"directive_id=(dir_[a-f0-9]{8})", sticky.stdout)
        assert sticky_id_match
        sticky_id = sticky_id_match.group(1)

        inspect_sticky = run(f"{binary} session inspect {session_id}", cwd=tmp_path)
        assert inspect_sticky.returncode == 0, inspect_sticky.stderr
        inspect_sticky_doc = json.loads(inspect_sticky.stdout)

        lineage_a = None
        for lineage in inspect_sticky_doc["lineages"].values():
            if lineage["name"] == "A":
                lineage_a = lineage
                break

        assert lineage_a is not None
        assert lineage_a["directives"]["sticky"][0]["text"] == "be friendly"

        oneshot = run(
            f'{binary} directive set {session_id} B --text "fix typo" --oneshot',
            cwd=tmp_path,
        )
        assert oneshot.returncode == 0, oneshot.stderr

        inspect_oneshot = run(f"{binary} session inspect {session_id}", cwd=tmp_path)
        assert inspect_oneshot.returncode == 0, inspect_oneshot.stderr
        inspect_oneshot_doc = json.loads(inspect_oneshot.stdout)

        lineage_b = None
        for lineage in inspect_oneshot_doc["lineages"].values():
            if lineage["name"] == "B":
                lineage_b = lineage
                break

        assert lineage_b is not None
        assert lineage_b["directives"]["oneshot"][0]["text"] == "fix typo"

        cleared = run(f"{binary} directive clear {session_id} A {sticky_id}", cwd=tmp_path)
        assert cleared.returncode == 0, cleared.stderr

        inspect_cleared = run(f"{binary} session inspect {session_id}", cwd=tmp_path)
        assert inspect_cleared.returncode == 0, inspect_cleared.stderr
        inspect_cleared_doc = json.loads(inspect_cleared.stdout)

        lineage_a = None
        for lineage in inspect_cleared_doc["lineages"].values():
            if lineage["name"] == "A":
                lineage_a = lineage
                break

        assert lineage_a is not None
        assert lineage_a["directives"]["sticky"] == []

        missing_type = run(
            f'{binary} directive set {session_id} A --text "test"',
            cwd=tmp_path,
        )
        assert missing_type.returncode != 0
        assert "must specify --oneshot or --sticky" in missing_type.stderr
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
