# instill

A Go library for installing [Agent Skills](https://agentskills.io/specification) into AI coding agents and detecting which agent is running your code.

40 agents can't agree on where to put a markdown file or which env var to set. instill deals with it so you don't have to.

```
your-tool/
└── skills/
    └── your-skill/
        ├── SKILL.md              # instructions + frontmatter
        └── references/           # optional detailed docs
            └── commands.md
```

No SDK, no runtime, no protocol. Markdown files in the right directories. Env vars in the right `if` statements.

> **Looking for a standalone tool to manage skills?** Use [vercel-labs/skills](https://github.com/vercel-labs/skills) instead. instill is a Go library for CLI tools that want to bundle and install their own agent skills as part of their distribution.

## Install skills

```go
import (
    "embed"
    "fmt"
    "log"

    "github.com/tiulpin/instill"
)

//go:embed skills
var skills embed.FS

func main() {
    // Detect which agents are present
    agents, _ := instill.Detect(".", false)
    names := make([]string, len(agents))
    for i, a := range agents {
        names[i] = a.Name
    }

    // Install skill files to each agent's expected location
    results, err := instill.Install(skills, instill.Options{
        Agents:     names,
        ProjectDir: ".",
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, r := range results {
        fmt.Printf("%s: %s (existed=%v)\n", r.Agent, r.Path, r.Existed)
    }
}
```

## Detect the running agent

```go
if agent := instill.DetectRuntime(); agent != nil {
    fmt.Printf("running inside %s (detected via %s)\n", agent.DisplayName, agent.EnvVar)
    // → "running inside Claude Code (detected via CLAUDECODE)"
}
```

Detection priority:
1. `AI_AGENT` — the emerging universal standard that everyone will adopt any day now
2. `AGENT` — used by Amp, Goose
3. `CLAUDE_CODE_IS_COWORK` — Claude Code cowork sessions
4. Agent-specific env vars (`CLAUDECODE`, `CURSOR_TRACE_ID`, `CODEX_SANDBOX`, etc.)
5. Filesystem markers (`/opt/.devin`)

Returns `nil` when no agent is detected (a.k.a. a human is typing).

## API

### Skill management

| Function                       | Description                                                                     |
|--------------------------------|---------------------------------------------------------------------------------|
| `Detect(projectDir, global)`   | Find which agents have config dirs present                                      |
| `Install(fsys, opts)`          | Copy skill files to each agent's skills directory                               |
| `Remove(name, opts)`           | Delete an installed skill by name                                               |
| `InstalledVersion(name, opts)` | Read `version` from an installed skill's frontmatter; returns `(string, error)` |
| `SkillVersion(fsys)`           | Read `version` from a skill FS (e.g. embedded)                                  |
| `AgentNames()`                 | List all supported agent names                                                  |

### Runtime detection

| Function          | Description                                                          |
|-------------------|----------------------------------------------------------------------|
| `DetectRuntime()` | Returns `*RuntimeAgent` for the agent running this process, or `nil` |

`RuntimeAgent` has three fields: `Name` (e.g. `"claude-code"`), `DisplayName` (e.g. `"Claude Code"`), and `EnvVar` (the variable that matched, e.g. `"CLAUDECODE"`).

## Upstream sync

Agent directories are sourced from [vercel-labs/skills](https://github.com/vercel-labs/skills/blob/main/src/agents.ts). Runtime env vars are cross-referenced with [vercel/vercel detect-agent](https://github.com/vercel/vercel/blob/main/packages/detect-agent/src/index.ts). A weekly agentic workflow keeps both in sync.

40 agents supported. Run `instill.AgentNames()` for the full list, or see [agents.go](agents.go).
