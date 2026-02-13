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
            "choices": [{"message": {"content": "evidence-pack-response"}}],
            "usage": {"prompt_tokens": 12, "completion_tokens": 7, "total_tokens": 19},
        }
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def log_message(self, format, *args):
        return


def _start_server() -> tuple[HTTPServer, threading.Thread]:
    server = HTTPServer(("127.0.0.1", 0), _MockOpenAIHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def test_us021_export_evidence_pack_json(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"
    server, thread = _start_server()

    try:
        build = run("go build -o ludus-magnus", cwd=repo_root)
        assert build.returncode == 0, build.stderr

        base_flags = (
            "--provider openai-compatible "
            "--api-key test-key "
            f"--base-url http://127.0.0.1:{server.server_port}"
        )

        init = run(f'{binary} quickstart init --need "evidence export" {base_flags}', cwd=tmp_path)
        assert init.returncode == 0, init.stderr
        session_match = re.search(r"session_id=(ses_[a-f0-9]{8})", init.stdout)
        assert session_match
        session_id = session_match.group(1)

        run_first = run(f'{binary} run {session_id} --input "first trial" {base_flags}', cwd=tmp_path)
        assert run_first.returncode == 0, run_first.stderr
        artifact_match = re.search(r"artifact_id=(art_[a-f0-9]{8})", run_first.stdout)
        assert artifact_match
        artifact_id = artifact_match.group(1)

        evaluate = run(
            f'{binary} evaluate {artifact_id} --score 8 --comment "good first pass"',
            cwd=tmp_path,
        )
        assert evaluate.returncode == 0, evaluate.stderr

        iterate = run(f'{binary} iterate {session_id} --lineage main {base_flags}', cwd=tmp_path)
        assert iterate.returncode == 0, iterate.stderr

        run_second = run(f'{binary} run {session_id} --input "second trial" {base_flags}', cwd=tmp_path)
        assert run_second.returncode == 0, run_second.stderr

        export_res = run(f"{binary} export evidence {session_id} --format json", cwd=tmp_path)
        assert export_res.returncode == 0, export_res.stderr

        evidence = json.loads(export_res.stdout)
        assert evidence["session_id"] == session_id
        assert evidence["mode"] == "quickstart"
        assert evidence["need"] == "evidence export"
        assert evidence["created_at"]
        assert len(evidence["lineages"]) == 1

        lineage = evidence["lineages"][0]
        assert lineage["name"] == "main"
        assert "agent_versions" in lineage
        assert len(lineage["agent_versions"]) >= 2
        assert "artifacts" in lineage
        assert len(lineage["artifacts"]) >= 2
        assert "directives" in lineage
        assert "oneshot" in lineage["directives"]
        assert "sticky" in lineage["directives"]

        evaluated_artifact = next(art for art in lineage["artifacts"] if art["id"] == artifact_id)
        assert evaluated_artifact["evaluation"]["score"] == 8
        assert evaluated_artifact["evaluation"]["comment"] == "good first pass"

        missing = run(f"{binary} export evidence ses_doesnotexist --format json", cwd=tmp_path)
        assert missing.returncode != 0
        assert 'session "ses_doesnotexist" not found' in (missing.stderr + missing.stdout)
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=1)
