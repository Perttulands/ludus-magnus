/**
 * Nudge-on-Idle — Headless extension for Chiron evaluations
 * 
 * On agent_end, checks if the agent made any edits. If not, sends a nudge
 * message telling it to use the edit tool. This addresses the "analysis
 * paralysis" failure mode where small models read and discuss but never act.
 * 
 * Headless-safe: no UI calls, works in -p --mode json.
 */

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  let editCount = 0;
  let turnCount = 0;
  let nudgeCount = 0;
  const MAX_NUDGES = 3; // Don't nudge forever

  // Track edit tool calls
  pi.on("tool_call", async (event, _ctx) => {
    if (event.toolName === "edit") {
      editCount++;
    }
    return { block: false };
  });

  // Count turns
  pi.on("turn_start", async (_event, _ctx) => {
    turnCount++;
  });

  // Nudge on agent_end if no edits after 3+ turns
  pi.on("agent_end", async (_event, _ctx) => {
    if (editCount === 0 && turnCount >= 3 && nudgeCount < MAX_NUDGES) {
      nudgeCount++;
      pi.sendMessage(
        {
          content: `You have completed ${turnCount} turns but made 0 edits. You must use the edit tool to fix the code. Read the source, find the bug, and edit the file directly. Do not describe what you would do — do it.`,
          display: false,
        },
        { triggerTurn: true },
      );
    }
  });
}
