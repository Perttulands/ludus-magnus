/**
 * Scan-Reminder — Headless extension for Chiron evaluations
 * 
 * Tracks which files the agent has read. On agent_end, if the agent has
 * made edits but hasn't read certain "should-check" files (like
 * debug_handler.go), sends a nudge to review them for security issues.
 * 
 * This targets B3 scoring — the incidental security finding that small
 * models often miss because they stop after fixing the primary bug.
 * 
 * Headless-safe: no UI calls, works in -p --mode json.
 */

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { isToolCallEventType } from "@mariozechner/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  let filesRead = new Set<string>();
  let editCount = 0;
  let nudged = false;

  // Track file reads
  pi.on("tool_call", async (event, _ctx) => {
    if (isToolCallEventType("read", event)) {
      filesRead.add(event.input.path);
    }
    if (event.toolName === "edit") {
      editCount++;
    }
    return { block: false };
  });

  // On agent_end, check if security-relevant files were reviewed
  pi.on("agent_end", async (_event, _ctx) => {
    if (nudged || editCount === 0) return;

    // Check if they read files beyond the main bug target
    const readDebugHandler = Array.from(filesRead).some(f => 
      f.includes("debug_handler")
    );

    if (!readDebugHandler) {
      nudged = true;
      pi.sendMessage(
        {
          content: `Good progress on the fix. Before finishing: read the other source files in the project (especially debug_handler.go) and check for security issues like exposed endpoints, missing auth, or leaked credentials. Use the read tool now.`,
          display: false,
        },
        { triggerTurn: true },
      );
    }
  });
}
