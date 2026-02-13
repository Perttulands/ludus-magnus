import json
import os
import subprocess
from pathlib import Path


GO_BIN = "/usr/local/go/bin/go"


def run(cmd, cwd):
    env = os.environ.copy()
    env["PATH"] = f"/usr/local/go/bin:{env.get('PATH', '')}"
    return subprocess.run(
        cmd,
        cwd=cwd,
        env=env,
        shell=True,
        text=True,
        capture_output=True,
        check=False,
    )


def test_us027_state_migration_framework_and_legacy_upgrade():
    repo_root = Path(__file__).resolve().parents[1]

    result = run(f"{GO_BIN} test ./internal/state -v", repo_root)
    assert result.returncode == 0, (
        "go test ./internal/state -v failed:\n"
        f"STDOUT:\n{result.stdout}\nSTDERR:\n{result.stderr}"
    )

    migration_test = repo_root / "internal" / "state" / "migration_test.go"
    assert migration_test.exists()
    assert migration_test.stat().st_size > 0

    state_dir = repo_root / ".ludus-magnus"
    state_dir.mkdir(exist_ok=True)
    state_file = state_dir / "state.json"
    state_file.write_text(
        json.dumps({"version": "0.9", "sessions": {}}, indent=2) + "\n",
        encoding="utf-8",
    )

    loader = """
package main

import (
  "fmt"
  "github.com/Perttulands/ludus-magnus/internal/state"
)

func main() {
  st, err := state.Load("")
  if err != nil {
    panic(err)
  }
  fmt.Println(st.Version)
}
"""
    loader_file = repo_root / ".ludus-magnus" / "tmp_load_state_main.go"
    loader_file.write_text(loader, encoding="utf-8")
    try:
        run_result = run(f"{GO_BIN} run {loader_file}", repo_root)
        assert run_result.returncode == 0, (
            "go run state loader failed:\n"
            f"STDOUT:\n{run_result.stdout}\nSTDERR:\n{run_result.stderr}"
        )
        assert run_result.stdout.strip() == "1.0"
    finally:
        loader_file.unlink(missing_ok=True)
