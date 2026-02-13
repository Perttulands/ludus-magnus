import subprocess
from pathlib import Path


def run(cmd: str) -> subprocess.CompletedProcess[str]:
    go_path_prefix = 'export PATH="/usr/local/go/bin:$PATH"; '
    return subprocess.run(
        ["bash", "-lc", f"{go_path_prefix}{cmd}"], capture_output=True, text=True
    )


def test_us001_project_scaffolding_acceptance():
    repo_root = Path(__file__).resolve().parents[1]

    tidy = run("go mod tidy")
    assert tidy.returncode == 0, tidy.stderr

    build = run("go build -o ludus-magnus")
    assert build.returncode == 0, build.stderr

    help_out = run("./ludus-magnus --help")
    assert help_out.returncode == 0, help_out.stderr
    assert "ludus-magnus" in help_out.stdout

    required_dirs = [
        repo_root / "internal" / "state",
        repo_root / "internal" / "provider",
        repo_root / "internal" / "engine",
        repo_root / "internal" / "export",
    ]
    for path in required_dirs:
        assert path.exists() and path.is_dir(), f"missing directory: {path}"
