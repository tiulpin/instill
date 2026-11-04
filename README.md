# instill

instill is a Go library for CLI tools that ship their own [Agent Skills](https://agentskills.io/specification). It installs skills into the coding agents found on the machine, can serve their pages from your binary instead, and reports which agent is running your process.

Every agent keeps skills in a different directory and sets different environment variables. instill keeps that table so your CLI doesn't have to.

To install skills from git repositories, use [vercel-labs/skills](https://github.com/vercel-labs/skills) instead.

## Skill layout

```
your-tool/
└── skills/
    └── your-skill/
        ├── SKILL.md          # frontmatter and instructions
        ├── references/       # pages served by Guide
        │   └── commands.md
        ├── scripts/          # copied on install
        ├── assets/           # copied on install
        └── _agents/          # Claude Code subagents
```

`SKILL.md` needs `name` and `description` in its frontmatter. `version`, at the top level or under `metadata`, is what `InstalledVersion` reports. A skill with `metadata.internal: true` is left out of `ListSkills` and installed only when `Options.Skills` names it.

## Install skills

```go
//go:embed skills
var skills embed.FS

func install() error {
    agents, err := instill.Detect(".", false)
    if err != nil {
        return err
    }
    names := make([]string, len(agents))
    for i, a := range agents {
        names[i] = a.Name
    }

    results, err := instill.Install(skills, instill.Options{
        Agents:     names,
        ProjectDir: ".",
    })
    if err != nil {
        return err
    }
    for _, r := range results {
        fmt.Printf("%s: %s (existed=%v)\n", r.Agent, r.Path, r.Existed)
    }
    return nil
}
```

`Install` copies each skill into the agents' skill directories. For Claude Code, files in `_commands/` and `_agents/` become slash commands and subagents. Agents that share a directory get one copy. Reinstalling replaces the skill directory and deletes commands and subagents the skill no longer ships. `Remove` deletes the skill along with its commands and subagents.

Set `Global` to install into each agent's home directory instead of the project.

## Serve the guide from the binary

A copied skill keeps the pages of the version that installed it until the next install. To keep the instructions in step with the binary, set `Command` and add a `skill guide <skill> [topic]` command to your CLI:

```go
instill.Install(skills, instill.Options{Agents: names, ProjectDir: ".", Command: "your-tool"})

// your-tool skill guide <skill> [topic]
page, err := instill.Guide(skills, "your-tool", skill, topic)
if err != nil {
    return err
}
fmt.Print(page)
```

With `Command` set, `Install` writes a short `SKILL.md` with the original frontmatter and a body that tells the agent to run `your-tool skill guide your-skill`. Markdown pages stay in the binary. Everything under `assets/` and `scripts/`, Markdown included, is copied along with any other non-Markdown file. Those copies only change on reinstall, so a skill that ships them should be reinstalled when `InstalledVersion` differs from `SkillVersion`.

Without a topic, `Guide` returns the `SKILL.md` body and a list of topics. A topic is a page path without `.md`, such as `references/commands`, or just `commands` if no other page has that name. Relative links to `.md` pages become the guide command for that page. An unknown topic returns an error that lists the valid ones, and `Topics` returns the same list for shell completion.

## Detect the running agent

```go
if agent := instill.DetectRuntime(); agent != nil {
    fmt.Printf("running inside %s (detected via %s)\n", agent.DisplayName, agent.EnvVar)
    // running inside Claude Code (detected via CLAUDECODE)
}
```

`DetectRuntime` checks, in order:

1. `AI_AGENT`, the cross-agent variable that everyone will adopt any day now. Names from vercel's detect-agent, such as `claude` or `github-copilot-cli`, map to instill's agent names.
2. `AGENT`, set by Amp and Goose
3. `CLAUDE_CODE_IS_COWORK`, set in Claude Code cowork sessions
4. agent-specific variables such as `CLAUDECODE`, `GEMINI_CLI` and `CODEX_SANDBOX`
5. Cursor: `CURSOR_EXTENSION_HOST_ROLE=agent-exec`, then `CURSOR_TRACE_ID`. Cursor's terminal sets the trace ID for every process, so it is checked last and an agent started from that terminal is reported as itself.
6. filesystem markers (`/opt/.devin`)

It returns `nil` when no agent matches, which usually means a human is typing.

## API

### Skills

| Function                         | Description                                                  |
|----------------------------------|--------------------------------------------------------------|
| `Install(fsys, opts)`            | Install skills, or stubs when `opts.Command` is set          |
| `Guide(fsys, cmd, skill, topic)` | Return a skill page for `<cmd> skill guide <skill> [topic]`  |
| `Topics(fsys, skill)`            | List a skill's pages, e.g. `references/commands`             |
| `ListSkills(fsys)`               | Name, version and description of each skill in `fsys`        |
| `SkillVersion(fsys)`             | Version of the first skill in `fsys`                         |
| `InstalledVersion(name, opts)`   | Version of an installed skill, or `""` if it isn't installed |
| `Remove(name, opts)`             | Delete an installed skill with its commands and subagents    |

### Agents

| Function                     | Description                                                   |
|------------------------------|---------------------------------------------------------------|
| `Detect(projectDir, global)` | Agents whose config directories exist                         |
| `AgentNames()`               | All supported agent names, sorted                             |
| `DetectRuntime()`            | The agent running this process as a `*RuntimeAgent`, or `nil` |

`RuntimeAgent` has `Name` (`"claude-code"`), `DisplayName` (`"Claude Code"`) and `EnvVar`, the variable that matched (`"CLAUDECODE"`).

## Agent registry

Skill directories come from [vercel-labs/skills](https://github.com/vercel-labs/skills/blob/main/src/agents.ts) and runtime variables from [vercel/vercel detect-agent](https://github.com/vercel/vercel/blob/main/packages/detect-agent/src/index.ts). A weekly workflow syncs both. The full list is in [agents.go](agents.go).
