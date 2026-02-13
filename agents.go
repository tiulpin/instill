package instill

import "sort"

type agent struct {
	name, displayName string
	skillsDir         string   // project-level, relative to project root
	globalDir         string   // global path template (before env/~ resolution)
	detectDirs        []string // dirs to check for agent presence
}

var agents = [...]agent{
	{"adal", "AdaL", ".adal/skills", "~/.adal/skills", d("~/.adal")},
	{"amp", "Amp", ".agents/skills", "$XDG_CONFIG_HOME/agents/skills", d("$XDG_CONFIG_HOME/amp")},
	{"antigravity", "Antigravity", ".agent/skills", "~/.gemini/antigravity/skills", d(".agent", "~/.gemini/antigravity")},
	{"augment", "Augment", ".augment/skills", "~/.augment/skills", d("~/.augment")},
	{"claude-code", "Claude Code", ".claude/skills", "$CLAUDE_CONFIG_DIR/skills", d("$CLAUDE_CONFIG_DIR")},
	{"cline", "Cline", ".cline/skills", "~/.cline/skills", d("~/.cline")},
	{"codebuddy", "CodeBuddy", ".codebuddy/skills", "~/.codebuddy/skills", d(".codebuddy", "~/.codebuddy")},
	{"codex", "Codex", ".agents/skills", "$CODEX_HOME/skills", d("$CODEX_HOME", "/etc/codex")},
	{"command-code", "Command Code", ".commandcode/skills", "~/.commandcode/skills", d("~/.commandcode")},
	{"continue", "Continue", ".continue/skills", "~/.continue/skills", d(".continue", "~/.continue")},
	{"crush", "Crush", ".crush/skills", "~/.config/crush/skills", d("~/.config/crush")},
	{"cursor", "Cursor", ".cursor/skills", "~/.cursor/skills", d("~/.cursor")},
	{"droid", "Droid", ".factory/skills", "~/.factory/skills", d("~/.factory")},
	{"gemini-cli", "Gemini CLI", ".agents/skills", "~/.gemini/skills", d("~/.gemini")},
	{"github-copilot", "GitHub Copilot", ".agents/skills", "~/.copilot/skills", d(".github", "~/.copilot")},
	{"goose", "Goose", ".goose/skills", "$XDG_CONFIG_HOME/goose/skills", d("$XDG_CONFIG_HOME/goose")},
	{"iflow-cli", "iFlow CLI", ".iflow/skills", "~/.iflow/skills", d("~/.iflow")},
	{"junie", "Junie", ".junie/skills", "~/.junie/skills", d("~/.junie")},
	{"kilo", "Kilo Code", ".kilocode/skills", "~/.kilocode/skills", d("~/.kilocode")},
	{"kimi-cli", "Kimi Code CLI", ".agents/skills", "~/.config/agents/skills", d("~/.kimi")},
	{"kiro-cli", "Kiro CLI", ".kiro/skills", "~/.kiro/skills", d("~/.kiro")},
	{"kode", "Kode", ".kode/skills", "~/.kode/skills", d("~/.kode")},
	{"mcpjam", "MCPJam", ".mcpjam/skills", "~/.mcpjam/skills", d("~/.mcpjam")},
	{"mistral-vibe", "Mistral Vibe", ".vibe/skills", "~/.vibe/skills", d("~/.vibe")},
	{"mux", "Mux", ".mux/skills", "~/.mux/skills", d("~/.mux")},
	{"neovate", "Neovate", ".neovate/skills", "~/.neovate/skills", d("~/.neovate")},
	{"openclaw", "OpenClaw", "skills", "~/.openclaw/skills", d("~/.openclaw", "~/.clawdbot", "~/.moltbot")},
	{"opencode", "OpenCode", ".agents/skills", "$XDG_CONFIG_HOME/opencode/skills", d("$XDG_CONFIG_HOME/opencode")},
	{"openhands", "OpenHands", ".openhands/skills", "~/.openhands/skills", d("~/.openhands")},
	{"pi", "Pi", ".pi/skills", "~/.pi/agent/skills", d("~/.pi/agent")},
	{"pochi", "Pochi", ".pochi/skills", "~/.pochi/skills", d("~/.pochi")},
	{"qoder", "Qoder", ".qoder/skills", "~/.qoder/skills", d("~/.qoder")},
	{"qwen-code", "Qwen Code", ".qwen/skills", "~/.qwen/skills", d("~/.qwen")},
	{"replit", "Replit", ".agents/skills", "$XDG_CONFIG_HOME/agents/skills", d(".agents")},
	{"roo", "Roo Code", ".roo/skills", "~/.roo/skills", d("~/.roo")},
	{"trae", "Trae", ".trae/skills", "~/.trae/skills", d("~/.trae")},
	{"trae-cn", "Trae CN", ".trae/skills", "~/.trae-cn/skills", d("~/.trae-cn")},
	{"windsurf", "Windsurf", ".windsurf/skills", "~/.codeium/windsurf/skills", d("~/.codeium/windsurf")},
	{"zencoder", "Zencoder", ".zencoder/skills", "~/.zencoder/skills", d("~/.zencoder")},
}

func d(dirs ...string) []string { return dirs }

var agentIndex = func() map[string]*agent {
	m := make(map[string]*agent, len(agents))
	for i := range agents {
		m[agents[i].name] = &agents[i]
	}
	return m
}()

// AgentNames returns all known agent names in sorted order.
func AgentNames() []string {
	names := make([]string, len(agents))
	for i := range agents {
		names[i] = agents[i].name
	}
	sort.Strings(names)
	return names
}
