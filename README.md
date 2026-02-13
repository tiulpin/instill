# instill

A Go library for installing [Agent Skills](https://agentskills.io/specification) into AI coding agents.

An agent skill is just a directory with a `SKILL.md` file — markdown instructions with YAML frontmatter. Agents like Claude Code, Cursor, and Windsurf each expect these files in different locations. instill puts them there.

```
your-tool/
└── skills/
    └── your-skill/
        ├── SKILL.md              # instructions + frontmatter
        └── references/           # optional detailed docs
            └── commands.md
```

That's it. No SDK, no runtime, no protocol. Markdown files in the right directories.

> **Looking for a standalone tool to manage skills?** Use [vercel-labs/skills](https://github.com/vercel-labs/skills) instead. instill is a Go library for CLI tools that want to bundle and install their own agent skills as part of their distribution.

## Usage

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

## API

| Function | Description |
|---|---|
| `Detect(projectDir, global)` | Find which agents have config dirs present |
| `Install(fsys, opts)` | Copy skill files to each agent's skills directory |
| `Remove(name, opts)` | Delete an installed skill by name |
| `InstalledVersion(name, opts)` | Read `version` from an installed skill's frontmatter; returns `(string, error)` |
| `SkillVersion(fsys)` | Read `version` from a skill FS (e.g. embedded) |
| `AgentNames()` | List all 39 supported agent names |

## What it does

1. Reads your `SKILL.md` frontmatter for the skill `name`
2. Resolves each agent's skills directory (project-local or global)
3. Cleans the target directory (removes stale files from prior installs)
4. Copies all skill files (`SKILL.md`, `references/`, etc.)
5. Reports what was created vs updated, with prior version from frontmatter

## Supported agents

39 agents supported. Paths sourced from [vercel-labs/skills](https://github.com/vercel-labs/skills/blob/main/src/agents.ts).

Run `instill.AgentNames()` for the full list, or see [agents.go](agents.go).
