# Agent_acadmey PRD

## 1. Summary

Agent_acadmey is a CLI-native system for training and evolving AI agents through iterative evaluation loops, designed for operation by code agents (Claude Code/Codex) rather than humans in a web UI.

## 2. Problem

Current Training Camp product value is strong, but delivery via web UI limits direct automation by coding agents. Teams need a programmable interface that preserves training semantics while enabling scripted and autonomous usage.

## 3. Product Goal

Provide a robust CLI that supports the full lifecycle:
1. define intent
2. generate agents
3. execute and evaluate artifacts
4. evolve agent lineages
5. export deployable definitions and evidence

## 4. Users

Primary users:
- AI coding agents acting on behalf of humans
- developers orchestrating training runs via scripts/CI

Secondary users:
- technical operators inspecting run history via terminal

## 5. Success Criteria

- Full quickstart and training loops runnable from CLI only
- Deterministic state persistence and replayability
- Machine-readable outputs suitable for autonomous chaining
- Export compatibility with downstream deployment environments

## 6. Core Concepts

- Session: container for one objective
- Mode: `quickstart` (single lineage) or `training` (four lineages)
- Lineage: persistent branch with lock/directive controls
- Agent Definition: versioned executable config
- Artifact: generated output from one run
- Evaluation: numeric score and optional feedback
- Cycle: run -> evaluate -> evolve

## 7. Functional Requirements

### FR-1 Session Management

System must support creating, listing, inspecting, and closing sessions.

### FR-2 Quickstart Flow

System must support immediate single-agent generation and iterative evolution from freeform feedback.

### FR-3 Training Flow

System must support four parallel lineages (A/B/C/D), scoring, lock/unlock, and regenerate-unlocked behavior.

### FR-4 Directives

System must support lineage-level directives as one-shot and sticky guidance.

### FR-5 Promotion

System must support promoting a quickstart session to training mode with strategy option:
- variations
- alternatives

### FR-6 Execution

System must execute agent definitions against provided input(s), with error visibility and metadata capture.

### FR-7 Evaluation

System must persist score/comment per artifact and enforce required scoring before training iteration.

### FR-8 Export

System must export:
- agent definitions (json/python/typescript)
- evidence packs (scores, comments, lineage history)

### FR-9 Doctor Checks

System must provide environment diagnostics, including:
- LLM configuration readiness
- availability of optional planning tools (including `beads` when installed)

### FR-10 Machine-Readable Output

All operational commands must support `--json` output.

## 8. Non-Functional Requirements

- Local-first operation
- Predictable command semantics
- No silent fallback behavior on execution errors
- Data durability across runs
- Extensible provider adapter layer

## 9. Out of Scope (v1)

- Browser UI
- Multi-user remote collaboration
- Hosted backend requirement
- Continuous online telemetry dependency

## 10. Data and Persistence

v1 persistence uses a local state file in workspace scope. Data model must retain all material required to reconstruct cycle history and export datasets.

## 11. Milestones

1. Docs and command contract
2. Minimal CLI scaffold with storage and session commands
3. Run/evaluate/iterate loop
4. Promotion/directives/locking
5. Export and doctor tooling
6. Stabilization and tests

## 12. Risks

- Prompt-evolution quality may be noisy without robust heuristics
- Provider behavior differences can create non-deterministic artifacts
- Large state files may become hard to manage without compaction strategy

## 13. Open Questions

- Should v1 default to JSON file or SQLite from day one?
- Should scoring be mandatory for every unlocked lineage before iteration?
- Which export format is the canonical interoperability target?
- How should multi-input test case libraries be prioritized in v1?

## 14. Additional Use Cases (Beyond Customer Care)

Customer care remains a primary target. The following use cases broaden applicability while preserving the same training loop semantics.

### UC-1 Incident Triage Agent (SRE/Platform)

Description:
- Classifies incoming incidents, enriches context from observability systems, and recommends first-response actions and escalation paths.

Typical tool calls:
- `get_service_health`
- `query_logs`
- `query_recent_deploys`
- `create_incident_ticket`

CLI flow:
1. `agent-acadmey session new --mode training --need "triage production incidents and propose next actions"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "good diagnosis, wrong priority level"`
4. `agent-acadmey directive set <session-id> B --sticky --text "prioritize user-impact over infra noise"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export agent <agent-id> --format typescript`

### UC-2 Sales Lead Qualification Agent

Description:
- Qualifies inbound leads using CRM and enrichment data, then recommends routing, follow-up timing, and messaging style.

Typical tool calls:
- `lookup_company_profile`
- `lookup_crm_history`
- `score_lead_fit`
- `create_sales_task`

CLI flow:
1. `agent-acadmey session new --mode training --need "qualify and route inbound B2B leads"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "routing is right, follow-up too generic"`
4. `agent-acadmey directive set <session-id> C --oneshot --text "personalize first touch using CRM context"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export evidence <session-id> --format json`

### UC-3 HR Policy and Case Intake Agent

Description:
- Handles employee policy questions and case intake with compliant responses and escalation when requests require human HR review.

Typical tool calls:
- `search_policy_docs`
- `check_employee_region_rules`
- `open_hr_case`
- `notify_hr_partner`

CLI flow:
1. `agent-acadmey session new --mode training --need "handle leave and benefits policy intake"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "policy cited correctly, escalation threshold unclear"`
4. `agent-acadmey directive set <session-id> A --sticky --text "escalate legal-sensitive requests immediately"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export agent <agent-id> --format json`

### UC-4 Internal IT Helpdesk Agent

Description:
- Resolves common IT requests (access, resets, setup issues) using identity and device-management tool calls with auditability.

Typical tool calls:
- `get_identity_status`
- `reset_mfa`
- `check_device_compliance`
- `create_it_ticket`

CLI flow:
1. `agent-acadmey session new --mode training --need "automate tier-1 IT helpdesk workflows"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "good remediation, missed compliance warning"`
4. `agent-acadmey directive set <session-id> D --sticky --text "always run compliance check before account actions"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export evidence <session-id> --format json`

### UC-5 Finance Invoice Exception Agent

Description:
- Reviews invoice exceptions, validates PO alignment, and proposes resolution actions with controllership-friendly traceability.

Typical tool calls:
- `get_invoice`
- `get_purchase_order`
- `match_invoice_to_po`
- `create_ap_exception_case`

CLI flow:
1. `agent-acadmey session new --mode training --need "resolve AP invoice exceptions with policy checks"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "strong reconciliation, weak explanation for rejection"`
4. `agent-acadmey directive set <session-id> B --oneshot --text "include plain-language reason codes in responses"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export agent <agent-id> --format python`

### UC-6 Contract Intake and Review Prep Agent

Description:
- Performs first-pass contract intake, extracts key clauses and risk flags, and routes items to legal owners with summaries.

Typical tool calls:
- `parse_contract_document`
- `check_clause_library`
- `flag_risk_terms`
- `open_legal_review_task`

CLI flow:
1. `agent-acadmey session new --mode training --need "triage vendor contracts for legal review"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "good extraction, indemnity risk underweighted"`
4. `agent-acadmey directive set <session-id> C --sticky --text "prioritize indemnity and liability deviations"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export evidence <session-id> --format json`

### UC-7 Prior Authorization Coordination Agent (Healthcare Ops)

Description:
- Coordinates prior authorization workflows for operations teams (administrative support only, no diagnosis), ensuring required documentation is complete.

Typical tool calls:
- `check_payer_requirements`
- `verify_patient_eligibility`
- `validate_required_docs`
- `create_prior_auth_submission`

CLI flow:
1. `agent-acadmey session new --mode training --need "prepare prior auth submissions for specialty procedures"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "good checklist, missed payer-specific form rule"`
4. `agent-acadmey directive set <session-id> A --sticky --text "strictly separate guidance by payer plan"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export agent <agent-id> --format typescript`

### UC-8 Logistics Shipment Exception Agent

Description:
- Handles shipment delays, address failures, and carrier exceptions by synthesizing status data and proposing remediation actions.

Typical tool calls:
- `track_shipment`
- `check_carrier_events`
- `estimate_recovery_eta`
- `create_logistics_case`

CLI flow:
1. `agent-acadmey session new --mode training --need "manage shipment exceptions and recovery communication"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "accurate status, weak fallback options"`
4. `agent-acadmey directive set <session-id> D --oneshot --text "always provide two recovery options"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export evidence <session-id> --format json`

### UC-9 Engineering Runbook Execution Agent

Description:
- Translates runbooks into reliable operational actions with explicit prechecks, guardrails, and escalation triggers.

Typical tool calls:
- `read_runbook`
- `check_preconditions`
- `execute_safe_command`
- `open_escalation`

CLI flow:
1. `agent-acadmey session new --mode training --need "execute standard runbooks for service recovery"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "great sequencing, insufficient rollback guidance"`
4. `agent-acadmey directive set <session-id> B --sticky --text "include rollback and stop conditions in every plan"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export agent <agent-id> --format json`

### UC-10 Knowledge Base Curation Agent

Description:
- Consolidates repeated tickets/chats into publishable knowledge articles and identifies stale or conflicting documentation.

Typical tool calls:
- `search_support_history`
- `cluster_similar_issues`
- `draft_kb_article`
- `publish_kb_update`

CLI flow:
1. `agent-acadmey session new --mode training --need "generate high-quality KB articles from repeated issues"`
2. `agent-acadmey run <session-id>`
3. `agent-acadmey evaluate <artifact-id> --score 1-10 --comment "good article structure, weak troubleshooting depth"`
4. `agent-acadmey directive set <session-id> C --sticky --text "include diagnostics and verification steps"`
5. `agent-acadmey iterate <session-id>`
6. `agent-acadmey export evidence <session-id> --format json`
