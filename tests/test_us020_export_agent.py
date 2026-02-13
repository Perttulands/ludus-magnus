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
            "choices": [{"message": {"content": "You are a robust helper."}}],
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


def test_us020_export_agent_formats_and_missing_agent_error(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"
    server, thread = _start_server()

    try:
        build = run("go build -o ludus-magnus", cwd=repo_root)
        assert build.returncode == 0, build.stderr

        init = run(
            (
                f'{binary} quickstart init --need "exportable agent" '
                f"--provider openai-compatible --api-key test-key --base-url http://127.0.0.1:{server.server_port}"
            ),
            cwd=tmp_path,
        )
        assert init.returncode == 0, init.stderr

        state_doc = json.loads((tmp_path / ".ludus-magnus" / "state.json").read_text())
        session_doc = next(iter(state_doc["sessions"].values()))
        lineage_doc = next(iter(session_doc["lineages"].values()))
        lineage_doc["agents"][0]["definition"]["tools"] = [
            {"name": "search", "type": "function"},
            {"name": "calculator", "type": "function"},
        ]
        (tmp_path / ".ludus-magnus" / "state.json").write_text(json.dumps(state_doc, indent=2) + "\n")
        agent_id = lineage_doc["agents"][0]["id"]

        exported_json = run(f"{binary} export agent {agent_id} --format json", cwd=tmp_path)
        assert exported_json.returncode == 0, exported_json.stderr
        agent_def = json.loads(exported_json.stdout)
        assert agent_def["system_prompt"] == "You are a robust helper."
        assert "model" in agent_def
        assert "temperature" in agent_def
        assert "max_tokens" in agent_def
        assert "tools" in agent_def

        exported_python = run(f"{binary} export agent {agent_id} --format python > agent.py", cwd=tmp_path)
        assert exported_python.returncode == 0, exported_python.stderr
        import_python = run('python3 -c "import agent; print(agent.agent_definition.get(\'model\'))"', cwd=tmp_path)
        assert import_python.returncode == 0, import_python.stderr
        assert import_python.stdout.strip() == agent_def["model"]
        import_python_tools = run(
            'python3 -c "import agent; print(len(agent.agent_definition.get(\'tools\', [])))"',
            cwd=tmp_path,
        )
        assert import_python_tools.returncode == 0, import_python_tools.stderr
        assert import_python_tools.stdout.strip() == "2"

        exported_typescript = run(f"{binary} export agent {agent_id} --format typescript > agent.ts", cwd=tmp_path)
        assert exported_typescript.returncode == 0, exported_typescript.stderr
        ts_code = (tmp_path / "agent.ts").read_text()
        assert "const agentDefinition: AgentDefinition" in ts_code
        assert "systemPrompt" in ts_code
        assert "maxTokens" in ts_code
        assert '"name":"search"' in ts_code
        assert '"name":"calculator"' in ts_code

        missing = run(f"{binary} export agent nonexistent-id --format json", cwd=tmp_path)
        assert missing.returncode != 0
        assert 'agent "nonexistent-id" not found' in (missing.stderr + missing.stdout)
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
