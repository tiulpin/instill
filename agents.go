package instill

import "slices"

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
	{"aider-desk", "AiderDesk", ".aider-desk/skills", "~/.aider-desk/skills", d("~/.aider-desk"), nil},
	{"amp", "Amp", ".agents/skills", "$XDG_CONFIG_HOME/agents/skills", d("$XDG_CONFIG_HOME/amp"), e("AGENT")},
	{"antigravity", "Antigravity", ".agents/skills", "~/.gemini/config/skills", d("~/.gemini/antigravity"), e("ANTIGRAVITY_AGENT")},
	{"antigravity-cli", "Antigravity CLI", ".agents/skills", "~/.gemini/config/skills", d("~/.gemini/antigravity-cli"), nil},
	{"astrbot", "AstrBot", "data/skills", "~/.astrbot/data/skills", d("data/skills", "~/.astrbot"), nil},
	{"augment", "Augment", ".augment/skills", "~/.augment/skills", d("~/.augment"), e("AUGMENT_AGENT")},
	{"autohand-code", "Autohand Code CLI", ".autohand/skills", "$AUTOHAND_HOME/skills", d("$AUTOHAND_HOME"), nil},
	{"bob", "IBM Bob", ".bob/skills", "~/.bob/skills", d("~/.bob"), nil},
	{"claude-code", "Claude Code", ".claude/skills", "$CLAUDE_CONFIG_DIR/skills", d("$CLAUDE_CONFIG_DIR"), e("CLAUDECODE", "CLAUDE_CODE")},
	{"cline", "Cline", ".agents/skills", "~/.agents/skills", d("~/.cline"), e("CLINE_ACTIVE")},
	{"codearts-agent", "CodeArts Agent", ".codeartsdoer/skills", "~/.codeartsdoer/skills", d("~/.codeartsdoer"), nil},
	{"codebuddy", "CodeBuddy", ".codebuddy/skills", "~/.codebuddy/skills", d(".codebuddy", "~/.codebuddy"), nil},
	{"codemaker", "Codemaker", ".codemaker/skills", "~/.codemaker/skills", d("~/.codemaker"), nil},
	{"codestudio", "Code Studio", ".codestudio/skills", "~/.codestudio/skills", d("~/.codestudio"), nil},
	{"codex", "Codex", ".agents/skills", "~/.agents/skills", d("$CODEX_HOME", "/etc/codex"), e("CODEX_SANDBOX", "CODEX_CI", "CODEX_THREAD_ID")},
	{"command-code", "Command Code", ".commandcode/skills", "~/.commandcode/skills", d("~/.commandcode"), nil},
	{"continue", "Continue", ".continue/skills", "~/.continue/skills", d(".continue", "~/.continue"), nil},
	{"cortex", "Cortex Code", ".cortex/skills", "~/.snowflake/cortex/skills", d("~/.snowflake/cortex"), nil},
	{"crush", "Crush", ".crush/skills", "~/.config/crush/skills", d("~/.config/crush"), nil},
	{"cursor", "Cursor", ".agents/skills", "~/.cursor/skills", d("~/.cursor"), e("CURSOR_AGENT")},
	{"deepagents", "Deep Agents", ".agents/skills", "~/.deepagents/agent/skills", d("~/.deepagents"), nil},
	{"devin", "Devin for Terminal", ".devin/skills", "$XDG_CONFIG_HOME/devin/skills", d("$XDG_CONFIG_HOME/devin"), nil},
	{"dexto", "Dexto", ".agents/skills", "~/.agents/skills", d("~/.dexto"), nil},
	{"droid", "Droid", ".agents/skills", "~/.factory/skills", d("~/.factory"), nil},
	{"eve", "Eve", "agent/skills", "", nil, nil},
	{"firebender", "Firebender", ".agents/skills", "~/.firebender/skills", d("~/.firebender"), nil},
	{"forgecode", "ForgeCode", ".forge/skills", "~/.forge/skills", d("~/.forge"), nil},
	{"fx", "fx", ".agents/skills", "~/.agents/skills", d("~/.fx"), nil},
	{"gemini-cli", "Gemini CLI", ".agents/skills", "~/.gemini/skills", d("~/.gemini"), e("GEMINI_CLI")},
	{"github-copilot", "GitHub Copilot", ".agents/skills", "~/.copilot/skills", d("~/.copilot"), e("COPILOT_MODEL", "COPILOT_ALLOW_ALL", "COPILOT_GITHUB_TOKEN", "COPILOT_CLI")},
	{"goose", "Goose", ".goose/skills", "$XDG_CONFIG_HOME/goose/skills", d("$XDG_CONFIG_HOME/goose"), e("GOOSE_TERMINAL")},
	{"grok", "Grok Build", ".grok/skills", "$GROK_HOME/skills", d("$GROK_HOME"), nil},
	{"hermes-agent", "Hermes Agent", ".hermes/skills", "$HERMES_HOME/skills", d("$HERMES_HOME"), nil},
	{"iflow-cli", "iFlow CLI", ".iflow/skills", "~/.iflow/skills", d("~/.iflow"), nil},
	{"inference-sh", "inference.sh", ".inferencesh/skills", "~/.inferencesh/skills", d("~/.inferencesh"), nil},
	{"jazz", "Jazz", ".jazz/skills", "~/.jazz/skills", d("~/.jazz", ".jazz"), nil},
	{"junie", "Junie", ".junie/skills", "~/.junie/skills", d("~/.junie"), e("JUNIE")},
	{"kilo", "Kilo Code", ".agents/skills", "~/.kilo/skills", d("~/.kilo", "~/.kilocode"), nil},
	{"kimchi", "Kimchi", ".kimchi/skills", "~/.config/kimchi/harness/skills", d("~/.config/kimchi"), nil},
	{"kimi-code-cli", "Kimi Code CLI", ".agents/skills", "~/.agents/skills", d("~/.kimi-code", "~/.kimi"), nil},
	{"kiro-cli", "Kiro CLI", ".kiro/skills", "~/.kiro/skills", d("~/.kiro"), e("KIRO")},
	{"kode", "Kode", ".kode/skills", "~/.kode/skills", d("~/.kode"), nil},
	{"lingma", "Lingma", ".lingma/skills", "~/.lingma/skills", d("~/.lingma"), nil},
	{"loaf", "Loaf", ".agents/skills", "~/.agents/skills", d("~/.loaf"), nil},
	{"mcpjam", "MCPJam", ".mcpjam/skills", "~/.mcpjam/skills", d("~/.mcpjam"), nil},
	{"minimax-code", "MiniMax Code", ".minimax/skills", "~/.minimax/skills", d("~/.minimax", "/Applications/MiniMax Code.app"), nil},
	{"mistral-vibe", "Mistral Vibe", ".vibe/skills", "$VIBE_HOME/skills", d("$VIBE_HOME"), nil},
	{"moxby", "Moxby", ".moxby/skills", "~/.moxby/skills", d("~/.moxby"), nil},
	{"mux", "Mux", ".mux/skills", "~/.mux/skills", d("~/.mux"), nil},
	{"neovate", "Neovate", ".neovate/skills", "~/.neovate/skills", d("~/.neovate"), nil},
	{"ona", "Ona", ".ona/skills", "~/.ona/skills", d("~/.ona"), nil},
	{"openclaw", "OpenClaw", "skills", "~/.openclaw/skills", d("~/.openclaw", "~/.clawdbot", "~/.moltbot"), e("OPENCLAW_SHELL")},
	{"opencode", "OpenCode", ".agents/skills", "$XDG_CONFIG_HOME/opencode/skills", d("$XDG_CONFIG_HOME/opencode"), e("OPENCODE_CLIENT", "OPENCODE")},
	{"openhands", "OpenHands", ".openhands/skills", "~/.openhands/skills", d("~/.openhands"), nil},
	{"pi", "Pi", ".agents/skills", "~/.agents/skills", d("~/.pi/agent"), nil},
	{"pochi", "Pochi", ".pochi/skills", "~/.pochi/skills", d("~/.pochi"), nil},
	{"posit-assistant", "Posit Assistant", ".posit/assistant/skills", "~/.posit/assistant/skills", d("~/.posit/assistant", "~/.positai"), nil},
	{"promptscript", "PromptScript", ".agents/skills", "", d(".promptscript", "promptscript.yaml"), nil},
	{"qoder", "Qoder", ".qoder/skills", "~/.qoder/skills", d("~/.qoder"), nil},
	{"qoder-cn", "Qoder CN", ".qoder/skills", "~/.qoder-cn/skills", d("~/.qoder-cn"), nil},
	{"qwen-code", "Qwen Code", ".qwen/skills", "~/.qwen/skills", d("~/.qwen"), nil},
	{"reasonix", "Reasonix", ".reasonix/skills", "$REASONIX_HOME/skills", d("$REASONIX_HOME"), nil},
	{"replit", "Replit", ".agents/skills", "$XDG_CONFIG_HOME/agents/skills", d(".replit"), e("REPL_ID")},
	{"roo", "Roo Code", ".roo/skills", "~/.roo/skills", d("~/.roo"), e("ROO_ACTIVE")},
	{"rovodev", "Rovo Dev", ".rovodev/skills", "~/.rovodev/skills", d("~/.rovodev"), nil},
	{"sarvam-code", "Sarvam Code", ".agents/skills", "~/.agents/skills", d("$SARVAM_HOME"), nil},
	{"tabnine-cli", "Tabnine CLI", ".tabnine/agent/skills", "~/.tabnine/agent/skills", d("~/.tabnine"), nil},
	{"terramind", "Terramind", ".terramind/skills", "~/.terramind/skills", d("~/.terramind"), nil},
	{"tinycloud", "Tinycloud", ".tinycloud/skills", "~/.tinycloud/skills", d("~/.tinycloud"), nil},
	{"trae", "Trae", ".trae/skills", "~/.trae/skills", d("~/.trae"), e("TRAE_AI_SHELL_ID")},
	{"trae-cn", "Trae CN", ".trae/skills", "~/.trae-cn/skills", d("~/.trae-cn"), e("TRAE_AI_SHELL_ID")},
	{"warp", "Warp", ".agents/skills", "~/.agents/skills", d("~/.warp"), nil},
	{"windsurf", "Windsurf", ".windsurf/skills", "~/.codeium/windsurf/skills", d("~/.codeium/windsurf"), nil},
	{"zcode", "ZCode", ".zcode/skills", "~/.zcode/skills", d("~/.zcode", "/Applications/ZCode.app"), nil},
	{"zed", "Zed", ".agents/skills", "~/.agents/skills", d("$XDG_CONFIG_HOME/zed", "$APPDATA/Zed", "$FLATPAK_XDG_CONFIG_HOME/zed"), nil},
	{"zencoder", "Zencoder", ".zencoder/skills", "~/.zencoder/skills", d("~/.zencoder"), nil},
	{"zenflow", "Zenflow", ".zencoder/skills", "~/.zencoder/skills", d("~/.zencoder"), nil},
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
	slices.Sort(names)
	return names
}
