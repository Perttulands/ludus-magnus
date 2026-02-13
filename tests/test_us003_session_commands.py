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


def test_us003_session_commands_acceptance(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"

    build = run("go build -o ludus-magnus", cwd=repo_root)
    assert build.returncode == 0, build.stderr

    new = run(
        f'{binary} session new --mode quickstart --need "test intent"',
        cwd=tmp_path,
    )
    assert new.returncode == 0, new.stderr

    session_id = new.stdout.strip()
    assert re.fullmatch(r"ses_[a-f0-9]{8}", session_id)

    listed = run(f"{binary} session list", cwd=tmp_path)
    assert listed.returncode == 0, listed.stderr
    header = listed.stdout.splitlines()[0]
    assert all(column in header for column in ["ID", "MODE", "STATUS", "CREATED_AT"])
    assert session_id in listed.stdout
    assert "quickstart" in listed.stdout
    assert "active" in listed.stdout

    inspect = run(f"{binary} session inspect {session_id}", cwd=tmp_path)
    assert inspect.returncode == 0, inspect.stderr
    details = json.loads(inspect.stdout)
    assert details["id"] == session_id
    assert details["mode"] == "quickstart"
    assert details["need"] == "test intent"

    state_file = tmp_path / ".ludus-magnus" / "state.json"
    assert state_file.exists()
    state_doc = json.loads(state_file.read_text())
    assert session_id in state_doc["sessions"]
    assert state_doc["sessions"][session_id]["mode"] == "quickstart"
    assert state_doc["sessions"][session_id]["need"] == "test intent"
