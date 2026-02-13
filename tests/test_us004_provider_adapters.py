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


def test_us004_provider_adapters_acceptance():
    repo_root = Path(__file__).resolve().parents[1]

    provider_tests = run("go test ./internal/provider -v", cwd=repo_root)
    assert provider_tests.returncode == 0, provider_tests.stderr
    assert "PASS" in provider_tests.stdout

    provider_build = run("go build ./internal/provider", cwd=repo_root)
    assert provider_build.returncode == 0, provider_build.stderr
