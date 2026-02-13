import json
import re
import subprocess
from pathlib import Path


def run(cmd: str, cwd: Path) -> subprocess.CompletedProcess[str]:
    go_path_prefix = 'export PATH="/usr/local/go/bin:$PATH"; '
    return subprocess.run(
        ["bash", "-lc", f"{go_path_prefix}{cmd}"],
        cwd=cwd,
        capture_output=True,
        text=True,
    )


def test_us005_quickstart_init_acceptance(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"

    build = run("go build -o ludus-magnus", cwd=repo_root)
    assert build.returncode == 0, build.stderr

    init = run(
        f'{binary} quickstart init --need "customer care agent"',
        cwd=tmp_path,
    )
    assert init.returncode == 0, init.stderr

    lines = [line.strip() for line in init.stdout.splitlines() if line.strip()]
    assert len(lines) == 2

    session_line, lineage_line = lines
    assert session_line.startswith("session_id=")
    assert lineage_line.startswith("lineage_id=")

    session_id = session_line.split("=", 1)[1]
    lineage_id = lineage_line.split("=", 1)[1]

    assert re.fullmatch(r"ses_[a-f0-9]{8}", session_id)
    assert re.fullmatch(r"lin_[a-f0-9]{8}", lineage_id)

    listed = run(f"{binary} session list", cwd=tmp_path)
    assert listed.returncode == 0, listed.stderr
    assert session_id in listed.stdout
    assert "quickstart" in listed.stdout

    state_file = tmp_path / ".ludus-magnus" / "state.json"
    assert state_file.exists()

    state_doc = json.loads(state_file.read_text())
    session = state_doc["sessions"][session_id]
    assert session["mode"] == "quickstart"
    assert session["need"] == "customer care agent"

    lineages = session["lineages"]
    assert lineage_id in lineages
    assert lineages[lineage_id]["name"] == "main"
