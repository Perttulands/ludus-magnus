import json
import os
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


def test_us023_global_json_output_flag(tmp_path: Path):
    repo_root = Path(__file__).resolve().parents[1]
    binary = repo_root / "ludus-magnus"

    build = run("go build -o ludus-magnus", cwd=repo_root)
    assert build.returncode == 0, build.stderr

    state_dir = tmp_path / ".ludus-magnus"
    state_dir.mkdir(parents=True, exist_ok=True)

    state_doc = {
        "version": "1.0",
        "sessions": {
            "ses_abc12345": {
                "id": "ses_abc12345",
                "mode": "quickstart",
                "need": "json output coverage",
                "created_at": "2026-02-13T10:00:00Z",
                "status": "active",
                "lineages": {
                    "lin_main123": {
                        "id": "lin_main123",
                        "session_id": "ses_abc12345",
                        "name": "main",
                        "locked": False,
                        "agents": [
                            {
                                "id": "agt_def1234",
                                "lineage_id": "lin_main123",
                                "version": 1,
                                "definition": {
                                    "system_prompt": "you are useful",
                                    "model": "gpt-4o-mini",
                                    "temperature": 1.0,
                                    "max_tokens": 4096,
                                    "tools": [],
                                },
                                "created_at": "2026-02-13T10:01:00Z",
                                "generation_metadata": {
                                    "provider": "openai-compatible",
                                    "model": "gpt-4o-mini",
                                    "tokens_used": 10,
                                    "duration_ms": 2,
                                    "cost_usd": 0.0,
                                },
                            }
                        ],
                        "artifacts": [
                            {
                                "id": "art_xyz9876",
                                "agent_id": "agt_def1234",
                                "input": "hi",
                                "output": "hello",
                                "created_at": "2026-02-13T10:02:00Z",
                                "execution_metadata": {
                                    "provider": "openai-compatible",
                                    "model": "gpt-4o-mini",
                                    "executor": "api",
                                    "tokens_input": 5,
                                    "tokens_output": 5,
                                    "duration_ms": 4,
                                    "cost_usd": 0.0,
                                    "tool_calls": [],
                                },
                                "evaluation": {
                                    "score": 8,
                                    "comment": "good",
                                    "created_at": "2026-02-13T10:03:00Z",
                                },
                            }
                        ],
                        "directives": {"oneshot": [], "sticky": []},
                    }
                },
            }
        },
    }
    (state_dir / "state.json").write_text(json.dumps(state_doc, indent=2) + "\n")

    session_list = run(f"{binary} session list --json", cwd=tmp_path)
    assert session_list.returncode == 0, session_list.stderr
    session_payload = json.loads(session_list.stdout)
    assert session_payload["sessions"][0]["id"] == "ses_abc12345"

    artifact_list = run(f"{binary} artifact list ses_abc12345 --json", cwd=tmp_path)
    assert artifact_list.returncode == 0, artifact_list.stderr
    artifact_payload = json.loads(artifact_list.stdout)
    assert artifact_payload["artifacts"][0]["id"] == "art_xyz9876"

    doctor = run(
        f"{binary} doctor --json",
        cwd=tmp_path,
        env={"ANTHROPIC_API_KEY": "sk-ant-test", "PATH": os.environ.get("PATH", "")},
    )
    assert doctor.returncode == 0, doctor.stderr
    doctor_payload = json.loads(doctor.stdout)
    assert "checks" in doctor_payload
    assert len(doctor_payload["checks"]) >= 3
