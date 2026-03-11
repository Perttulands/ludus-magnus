/**
 * Workflow-Gate — Headless extension for Chiron evaluations
 * 
 * Lightweight workflow enforcement via tool_call hooks:
 * - Blocks write tool (prefer edit for modifications)
 * - After 5+ edits, nudges to run tests if they haven't been run
 * 
 * This addresses two failure modes:
 * 1. Small models using write (full file replacement) instead of edit (surgical)
 * 2. Small models editing endlessly without testing
 * 
 * Headless-safe: no UI calls, works in -p --mode json.
 */

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { isToolCallEventType } from "@mariozechner/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  let editCount = 0;
  let testRun = false;
  let testNudged = false;

  pi.on("tool_call", async (event, _ctx) => {
    // Track edits
    if (event.toolName === "edit") {
      editCount++;
    }

    // Track test runs
    if (isToolCallEventType("bash", event)) {
      const cmd = event.input.command || "";
      if (cmd.includes("go test")) {
        testRun = true;
      }
    }

    // Block write tool for existing files — prefer edit for surgical changes.
    // Allow write for new files (the model might legitimately create test files).
    if (isToolCallEventType("write", event)) {
      const path = event.input.path || "";
      // Allow writing to new files (test files, docs, etc.)
      if (path.includes("_test.go") || path.includes("LESSONS") || path.includes("lessons") || path.includes(".md")) {
        return { block: false };
      }
      return {
        block: true,
        reason: "Use the edit tool instead of write for modifying existing source files. The edit tool makes surgical changes; write replaces the entire file which can introduce bugs. If you need to create a NEW file, use write with a descriptive filename.",
      };
    }

    return { block: false };
  });

  // After enough edits, remind to test
  pi.on("agent_end", async (_event, _ctx) => {
    if (editCount >= 3 && !testRun && !testNudged) {
      testNudged = true;
      pi.sendMessage(
        {
          content: `You've made ${editCount} edits but haven't run tests yet. Run: bash -c "go test ./..." to verify your changes work. Fix any failures before continuing.`,
          display: false,
        },
        { triggerTurn: true },
      );
    }
  });
}
