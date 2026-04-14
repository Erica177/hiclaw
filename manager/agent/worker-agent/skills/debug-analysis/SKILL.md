---
name: debug-analysis
description: Analyze and debug target Workers by syncing their workspace files, exporting Matrix room messages, and reviewing LLM session logs. Only available on DebugWorkers.
---

# Debug Analysis

You are a DebugWorker. Your job is to analyze and diagnose issues with target Workers.

## Target Workspaces

Target Workers' files are available at `~/debug-targets/<worker-name>/`. Always sync before reading to ensure you have the latest data.

### Available data per target
- `SOUL.md`, `AGENTS.md` — Agent personality and instructions
- `openclaw.json` — Runtime configuration (model, plugins, channels)
- `.openclaw/agents/main/sessions/*.jsonl` — LLM session logs (conversation history)
- `.openclaw/identity/` — Agent identity metadata
- `skills/` — Active skill definitions
- `config/mcporter.json` — MCP server configuration

## Commands

### Sync target workspaces (pull latest from centralized storage)

Sync all targets:
```bash
bash ~/skills/debug-analysis/scripts/sync-workspace.sh --all
```

Sync a specific target:
```bash
bash ~/skills/debug-analysis/scripts/sync-workspace.sh --worker <name>
```

### Export Matrix room messages

Export messages from a specific room (last 24 hours by default):
```bash
bash ~/skills/debug-analysis/scripts/export-matrix-messages.sh --room-id '<room_id>' --hours 24
```

Export messages from a room by name substring:
```bash
bash ~/skills/debug-analysis/scripts/export-matrix-messages.sh --room-name 'Worker' --hours 6
```

The output is JSONL format printed to stdout. Each line is a JSON object with fields: `event_id`, `type`, `sender`, `timestamp`, `time`, `body`.

**Note**: Matrix credentials must be configured in `~/debug-config.json` for message export to work. The homeserver URL is read from `openclaw.json`.

## Debugging Workflow

1. **Sync** target workspaces first to get latest state
2. **Read** `SOUL.md` and `AGENTS.md` to understand the target's role and instructions
3. **Review** LLM session logs (`.openclaw/agents/main/sessions/*.jsonl`) to trace conversation flow
4. **Export** Matrix messages to see inter-agent communication and human interactions
5. **Check** `openclaw.json` for misconfigurations (wrong model, missing plugins, etc.)
6. **Report** findings with evidence (specific log entries, message excerpts, config issues)
