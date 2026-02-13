from pathlib import Path


def test_us026_cli_usage_docs_exist_and_cover_commands():
    repo_root = Path(__file__).resolve().parents[1]
    doc_path = repo_root / "docs" / "CLI_USAGE.md"

    assert doc_path.exists(), "docs/CLI_USAGE.md must exist"
    content = doc_path.read_text(encoding="utf-8")

    assert "## Commands" in content
    assert "ludus-magnus session new" in content

    expected_commands = [
        "ludus-magnus session new",
        "ludus-magnus session list",
        "ludus-magnus session inspect",
        "ludus-magnus quickstart init",
        "ludus-magnus run",
        "ludus-magnus evaluate score",
        "ludus-magnus evaluate comment",
        "ludus-magnus iterate",
        "ludus-magnus training init",
        "ludus-magnus training iterate",
        "ludus-magnus lineage lock",
        "ludus-magnus lineage unlock",
        "ludus-magnus promote",
        "ludus-magnus directive set",
        "ludus-magnus directive clear",
        "ludus-magnus artifact list",
        "ludus-magnus artifact inspect",
        "ludus-magnus export agent",
        "ludus-magnus export evidence",
        "ludus-magnus doctor",
    ]
    for command in expected_commands:
        assert command in content, f"Missing command docs for: {command}"

    assert "### Quickstart Workflow" in content
    assert "### Training Workflow" in content
    assert "### Promotion Workflow" in content
    assert "--json" in content
    assert ".ludus-magnus/state.json" in content
