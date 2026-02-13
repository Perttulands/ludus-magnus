import os
import shutil
import subprocess
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


def test_us022_doctor_command_provider_executor_and_state_checks(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"

    build = run("go build -o ludus-magnus", cwd=repo_root)
    assert build.returncode == 0, build.stderr

    with_api_key_env = {
        "ANTHROPIC_API_KEY": "sk-ant-test",
        "PATH": os.environ.get("PATH", ""),
    }

    success = run(f"{binary} doctor", cwd=tmp_path, env=with_api_key_env)
    assert success.returncode == 0, success.stderr
    assert "✓ ANTHROPIC_API_KEY set" in success.stdout

    state_dir = tmp_path / ".ludus-magnus"
    state_dir.mkdir(parents=True, exist_ok=True)
    state_file = state_dir / "state.json"
    state_file.write_text('{"version":"1.0","sessions":{}}\n')

    with_state = run(f"{binary} doctor", cwd=tmp_path, env=with_api_key_env)
    assert with_state.returncode == 0, with_state.stderr
    assert "✓ State file readable: .ludus-magnus/state.json" in with_state.stdout

    missing_env = {"ANTHROPIC_API_KEY": "", "PATH": os.environ.get("PATH", "")}
    missing = run(f"{binary} doctor", cwd=tmp_path, env=missing_env)
    assert missing.returncode == 1
    assert "missing ANTHROPIC_API_KEY" in (missing.stdout + missing.stderr)

    codex_path = shutil.which("codex")
    if codex_path:
        assert "✓ codex binary found (optional)" in with_state.stdout
