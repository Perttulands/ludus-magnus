import json
import threading
import subprocess
from pathlib import Path
from http.server import BaseHTTPRequestHandler, HTTPServer


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

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        payload = {
            "choices": [{"message": {"content": "You are a reliable customer care agent."}}],
            "usage": {"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30},
        }
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def log_message(self, format, *args):
        return


def _start_server() -> tuple[HTTPServer, threading.Thread]:
    server = HTTPServer(("127.0.0.1", 0), _MockOpenAIHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def test_us006_agent_definition_generation(tmp_path: Path):
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
                f'{binary} quickstart init --need "customer care agent" '
                f'--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}'
            ),
            cwd=tmp_path,
        )
        assert init.returncode == 0, init.stderr

        state_doc = json.loads((tmp_path / ".ludus-magnus" / "state.json").read_text())
        session = next(iter(state_doc["sessions"].values()))
        lineage = next(iter(session["lineages"].values()))
        agent = lineage["agents"][0]

        definition = agent["definition"]
        assert definition["system_prompt"] != ""
        assert definition["model"] == "gpt-4o-mini"
        assert definition["temperature"] == 1.0
        assert definition["max_tokens"] == 4096
        assert definition["tools"] == []
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)

