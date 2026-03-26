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
    expires: 14
---

## Sync agent registry with vercel-labs/skills

This workflow checks if `agents.go` in this repository is up to date with the upstream agent registry at [vercel-labs/skills](https://github.com/vercel-labs/skills).

### Context

This Go library (`github.com/tiulpin/instill`) manages skill installation for AI coding agents. The canonical list of agents and their directory paths comes from `vercel-labs/skills` at `src/agents.ts`. Our copy lives in `agents.go` as a Go array of structs.

Each agent entry in `agents.go` looks like:

```go
{"agent-id", "Display Name", ".project/skills", "~/.global/skills", d("~/.detect-dir")},
```

The upstream TypeScript source (`src/agents.ts`) defines agents with fields: `id`, `name`, `skillsDir`, `globalSkillsDir`, and `detectDirs`.

### Instructions

1. **Fetch upstream**: Download `https://raw.githubusercontent.com/vercel-labs/skills/main/src/agents.ts` and parse the agent definitions.

2. **Read local**: Read `agents.go` and parse the current agent entries.

3. **Compare**: Identify:
   - New agents in upstream not present locally (by `id`)
   - Path changes (`skillsDir` or `globalSkillsDir`) for existing agents
   - Agents removed upstream (present locally but not upstream)
   - Skip the `universal` agent if it has `detectInstalled: async () => false` or similar — it's a meta-agent.

4. **If no changes needed**: Exit cleanly, do not create a PR.

5. **If changes are needed**: Update `agents.go`:
   - Add new agents in alphabetical order within the array
   - Update changed paths for existing agents
   - Remove agents that no longer exist upstream
   - Keep the array sorted alphabetically by agent ID
   - Match the existing code style exactly (tabs, spacing, the `d()` helper for detectDirs)
   - For new agents, infer `detectDirs` from the upstream source — typically the global config directory

6. **Update tests**: If agent path changes affect test expectations in `instill_test.go`, update those too. Run `go test ./... -race` to verify everything passes.

7. **Create a pull request** with the changes. The PR description should list what changed (new agents, updated paths, removed agents).
