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


def test_us024_quickstart_integration_go_test_passes():
    repo_root = Path(__file__).resolve().parents[1]
    result = run("go test ./test/integration -v", cwd=repo_root)
    assert result.returncode == 0, result.stderr
    assert "TestQuickstartFlowEndToEnd" in result.stdout
