---
on:
  schedule: weekly
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

tools:
  github:
    toolsets: [default]

network:
  allowed:
    - github
    - go

safe-outputs:
  create-pull-request:
    title-prefix: "[sync] "
    labels: [automation, upstream-sync]
    draft: false
---

## Sync agent registry with upstream sources

This workflow checks if `agents.go` in this repository is up to date with two upstream sources:

1. **Agent directories**: [vercel-labs/skills](https://github.com/vercel-labs/skills) at `src/agents.ts` — the canonical list of agents and their skill/config directory paths.
2. **Runtime env vars**: [vercel/vercel](https://github.com/vercel/vercel) at `packages/detect-agent/src/index.ts` — environment variables that agents set when running shell commands, used for runtime detection.

### Context

This Go library (`github.com/tiulpin/instill`) manages skill installation and runtime agent detection for AI coding agents. The agent registry in `agents.go` has two dimensions per agent:

```go
{"agent-id", "Display Name", ".project/skills", "~/.global/skills", d("~/.detect-dir"), e("ENV_VAR1", "ENV_VAR2")},
```

- Fields 1-5 (name, display, paths, detectDirs) come from `vercel-labs/skills`
- Field 6 (`runtimeEnvs` via `e()` helper) comes from `vercel/vercel` detect-agent + our own additions

### Instructions

1. **Fetch upstream sources**:
   - Download `https://raw.githubusercontent.com/vercel-labs/skills/main/src/agents.ts` and parse agent definitions (id, name, skillsDir, globalSkillsDir, detectDirs).
   - Download `https://raw.githubusercontent.com/vercel/vercel/main/packages/detect-agent/src/index.ts` and parse the runtime detection logic (env var → agent name mappings).

2. **Read local**: Read `agents.go` and parse the current agent entries, including `runtimeEnvs`.

3. **Compare agent directories** (from vercel-labs/skills):
   - New agents in upstream not present locally (by `id`)
   - Path changes (`skillsDir` or `globalSkillsDir`) for existing agents
   - Agents removed upstream (present locally but not upstream)
   - Skip the `universal` agent if it has `detectInstalled: async () => false` or similar — it's a meta-agent.

4. **Compare runtime env vars** (from vercel/vercel detect-agent):
   - New env var → agent mappings not present in our `runtimeEnvs`
   - Changed mappings (env var now maps to a different agent)
   - Removed mappings
   - Note: detect-agent also checks for `AI_AGENT` (universal) and `AGENT` (generic) — these are handled in `DetectRuntime()` logic, not in the per-agent `runtimeEnvs`. Do not add them to agent entries.
   - Note: detect-agent checks `CLAUDE_CODE_IS_COWORK` for cowork mode — this is handled as a special case in `DetectRuntime()`, not as a per-agent env var.
   - Note: detect-agent checks `/opt/.devin` filesystem path — this is handled in `DetectRuntime()` as a fallback, not in per-agent `runtimeEnvs`.

5. **If no changes needed**: Exit cleanly, do not create a PR.

6. **If changes are needed**: Update `agents.go`:
   - Add new agents in alphabetical order within the array
   - Update changed paths for existing agents
   - Update `runtimeEnvs` (`e()` helper) for changed env var mappings
   - For new agents from vercel-labs/skills that also appear in detect-agent, include their `runtimeEnvs`
   - For new agents without a known runtime env var, use `nil` for `runtimeEnvs`
   - Remove agents that no longer exist upstream
   - Keep the array sorted alphabetically by agent ID
   - Match the existing code style exactly (tabs, spacing, `d()` for detectDirs, `e()` for runtimeEnvs, `nil` when empty)

7. **Update tests**: If changes affect test expectations in `instill_test.go`, update those too. Run `go test ./... -race` to verify everything passes.

8. **Create a pull request** with the changes. The PR description should list what changed, grouped by source:
   - From vercel-labs/skills: new agents, updated paths, removed agents
   - From vercel/vercel detect-agent: new runtime env vars, updated mappings, removed mappings
