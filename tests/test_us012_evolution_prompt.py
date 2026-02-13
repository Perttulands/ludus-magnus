import os
import subprocess
from pathlib import Path


def run(cmd: str, cwd: Path) -> subprocess.CompletedProcess[str]:
    go_path_prefix = 'export PATH="/usr/local/go/bin:$PATH"; '
    return subprocess.run(
        ["bash", "-lc", f"{go_path_prefix}{cmd}"],
        cwd=cwd,
        capture_output=True,
        text=True,
        env=os.environ.copy(),
    )


def test_us012_evolution_prompt_generation():
    repo_root = Path(__file__).resolve().parents[1]

    result = run("go test ./internal/engine -run Evolution -v", cwd=repo_root)
    assert result.returncode == 0, result.stderr
    assert "TestGenerateEvolutionPromptIncludesFeedbackAndAverage" in result.stdout
    assert "TestGenerateEvolutionPromptWithoutEvaluationsStillGeneratesPrompt" in result.stdout
