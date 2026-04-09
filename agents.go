package instill

import "sort"

type agent struct {
	name, displayName string
	skillsDir         string
	globalDir         string
	detectDirs        []string
	runtimeEnvs       []string
}

func e(envs ...string) []string { return envs }

var agents = [...]agent{
	{"adal", "AdaL", ".adal/skills", "~/.adal/skills", d("~/.adal"), nil},
	{"amp", "Amp", ".agents/skills", "$XDG_CONFIG_HOME/agents/skills", d("$XDG_CONFIG_HOME/amp"), e("AGENT")},
	{"antigravity", "Antigravity", ".agents/skills", "~/.gemini/antigravity/skills", d("~/.gemini/antigravity"), e("ANTIGRAVITY_AGENT")},
	{"augment", "Augment", ".augment/skills", "~/.augment/skills", d("~/.augment"), e("AUGMENT_AGENT")},
	{"bob", "IBM Bob", ".bob/skills", "~/.bob/skills", d("~/.bob"), nil},
	{"claude-code", "Claude Code", ".claude/skills", "$CLAUDE_CONFIG_DIR/skills", d("$CLAUDE_CONFIG_DIR"), e("CLAUDECODE", "CLAUDE_CODE")},
	{"cline", "Cline", ".agents/skills", "~/.agents/skills", d("~/.cline"), e("CLINE_ACTIVE")},
	{"codebuddy", "CodeBuddy", ".codebuddy/skills", "~/.codebuddy/skills", d(".codebuddy", "~/.codebuddy"), nil},
	{"codex", "Codex", ".agents/skills", "$CODEX_HOME/skills", d("$CODEX_HOME", "/etc/codex"), e("CODEX_SANDBOX", "CODEX_CI", "CODEX_THREAD_ID")},
	{"command-code", "Command Code", ".commandcode/skills", "~/.commandcode/skills", d("~/.commandcode"), nil},
	{"continue", "Continue", ".continue/skills", "~/.continue/skills", d(".continue", "~/.continue"), nil},
	{"cortex", "Cortex Code", ".cortex/skills", "~/.snowflake/cortex/skills", d("~/.snowflake/cortex"), nil},
	{"crush", "Crush", ".crush/skills", "~/.config/crush/skills", d("~/.config/crush"), nil},
	{"cursor", "Cursor", ".agents/skills", "~/.cursor/skills", d("~/.cursor"), e("CURSOR_TRACE_ID", "CURSOR_AGENT")},
	{"deepagents", "Deep Agents", ".agents/skills", "~/.deepagents/agent/skills", d("~/.deepagents"), nil},
	{"droid", "Droid", ".factory/skills", "~/.factory/skills", d("~/.factory"), nil},
	{"firebender", "Firebender", ".agents/skills", "~/.firebender/skills", d("~/.firebender"), nil},
	{"gemini-cli", "Gemini CLI", ".agents/skills", "~/.gemini/skills", d("~/.gemini"), e("GEMINI_CLI")},
	{"github-copilot", "GitHub Copilot", ".agents/skills", "~/.copilot/skills", d(".github", "~/.copilot"), e("COPILOT_MODEL", "COPILOT_ALLOW_ALL", "COPILOT_GITHUB_TOKEN", "COPILOT_CLI")},
	{"goose", "Goose", ".goose/skills", "$XDG_CONFIG_HOME/goose/skills", d("$XDG_CONFIG_HOME/goose"), e("GOOSE_TERMINAL")},
	{"iflow-cli", "iFlow CLI", ".iflow/skills", "~/.iflow/skills", d("~/.iflow"), nil},
	{"junie", "Junie", ".junie/skills", "~/.junie/skills", d("~/.junie"), e("JUNIE")},
	{"kilo", "Kilo Code", ".kilocode/skills", "~/.kilocode/skills", d("~/.kilocode"), nil},
	{"kimi-cli", "Kimi Code CLI", ".agents/skills", "~/.config/agents/skills", d("~/.kimi"), nil},
	{"kiro-cli", "Kiro CLI", ".kiro/skills", "~/.kiro/skills", d("~/.kiro"), e("KIRO")},
	{"kode", "Kode", ".kode/skills", "~/.kode/skills", d("~/.kode"), nil},
	{"mcpjam", "MCPJam", ".mcpjam/skills", "~/.mcpjam/skills", d("~/.mcpjam"), nil},
	{"mistral-vibe", "Mistral Vibe", ".vibe/skills", "~/.vibe/skills", d("~/.vibe"), nil},
	{"mux", "Mux", ".mux/skills", "~/.mux/skills", d("~/.mux"), nil},
	{"neovate", "Neovate", ".neovate/skills", "~/.neovate/skills", d("~/.neovate"), nil},
	{"openclaw", "OpenClaw", "skills", "~/.openclaw/skills", d("~/.openclaw", "~/.clawdbot", "~/.moltbot"), e("OPENCLAW_SHELL")},
	{"opencode", "OpenCode", ".agents/skills", "$XDG_CONFIG_HOME/opencode/skills", d("$XDG_CONFIG_HOME/opencode"), e("OPENCODE_CLIENT", "OPENCODE")},
	{"openhands", "OpenHands", ".openhands/skills", "~/.openhands/skills", d("~/.openhands"), nil},
	{"pi", "Pi", ".pi/skills", "~/.pi/agent/skills", d("~/.pi/agent"), nil},
	{"pochi", "Pochi", ".pochi/skills", "~/.pochi/skills", d("~/.pochi"), nil},
	{"qoder", "Qoder", ".qoder/skills", "~/.qoder/skills", d("~/.qoder"), nil},
	{"qwen-code", "Qwen Code", ".qwen/skills", "~/.qwen/skills", d("~/.qwen"), nil},
	{"replit", "Replit", ".agents/skills", "$XDG_CONFIG_HOME/agents/skills", d(".agents"), e("REPL_ID")},
	{"roo", "Roo Code", ".roo/skills", "~/.roo/skills", d("~/.roo"), e("ROO_ACTIVE")},
	{"trae", "Trae", ".trae/skills", "~/.trae/skills", d("~/.trae"), e("TRAE_AI_SHELL_ID")},
	{"trae-cn", "Trae CN", ".trae/skills", "~/.trae-cn/skills", d("~/.trae-cn"), e("TRAE_AI_SHELL_ID")},
	{"warp", "Warp", ".agents/skills", "~/.agents/skills", d("~/.warp"), nil},
	{"windsurf", "Windsurf", ".windsurf/skills", "~/.codeium/windsurf/skills", d("~/.codeium/windsurf"), nil},
	{"zencoder", "Zencoder", ".zencoder/skills", "~/.zencoder/skills", d("~/.zencoder"), nil},
}

func d(dirs ...string) []string { return dirs }

var agentIndex = func() map[string]*agent {
	m := make(map[string]*agent, len(agents))
	for i := range agents {
		m[agents[i].name] = &agents[i]
	}
	return m
}()

// commandsDirs maps agent names to [project, global] command directories.
// Only agents with dedicated command file support need entries here.
var commandsDirs = map[string][2]string{
	"claude-code": {".claude/commands", "$CLAUDE_CONFIG_DIR/commands"},
}

// subagentsDirs maps agent names to [project, global] subagent directories.
var subagentsDirs = map[string][2]string{
	"claude-code": {".claude/agents", "$CLAUDE_CONFIG_DIR/agents"},
}

// AgentNames returns all known agent names in sorted order.
func AgentNames() []string {
	names := make([]string, len(agents))
	for i := range agents {
		names[i] = agents[i].name
	}
	sort.Strings(names)
	return names
}
