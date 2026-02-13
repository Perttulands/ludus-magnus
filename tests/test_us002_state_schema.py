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


def test_us002_state_schema_acceptance():
    repo_root = Path(__file__).resolve().parents[1]

    state_tests = run("go test ./internal/state -v", cwd=repo_root)
    assert state_tests.returncode == 0, state_tests.stderr
    assert "PASS" in state_tests.stdout

    state_build = run("go build ./internal/state", cwd=repo_root)
    assert state_build.returncode == 0, state_build.stderr

    save_default = run(
        "go test ./internal/state -run TestSaveUsesDefaultStatePath -v", cwd=repo_root
    )
    assert save_default.returncode == 0, save_default.stderr
    assert "PASS" in save_default.stdout
