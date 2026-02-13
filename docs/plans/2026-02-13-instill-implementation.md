# instill — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go library that installs AI agent skill files to the correct directories for 39 coding agents.

**Architecture:** Two files — `agents.go` (static data) and `instill.go` (everything else). Zero external dependencies. Frontmatter parsed with `bytes` stdlib — no YAML library needed for two key-value pairs.

**Tech Stack:** Go 1.22+, stdlib only (`io/fs`, `os`, `testing/fstest`)

**Design doc:** `../teamcity-cli/docs/plans/2026-02-13-instill-design.md`

**Target LOC:** ~350 total (agents data ~100, logic ~120, tests ~130)

---

### Task 1: Scaffold module + agent registry

**Files:**
- Create: `go.mod`
- Create: `agents.go`

**Step 1: Create the module**

```bash
mkdir -p /Users/tv/Projects/work/instill && cd /Users/tv/Projects/work/instill
git init
go mod init github.com/JetBrains/instill
```

**Step 2: Create agents.go**

This is the only file that's pure data. Every agent definition is a single struct — no separate "internal" vs "public" types.

Create `agents.go`:

```go
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
```

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: exit 0

**Step 4: Commit**

```bash
git add -A && git commit -m "init: module with agent registry (39 agents from vercel-labs/skills)"
```

---

### Task 2: Core API — Detect, Install, Remove

Everything in one file. Key design choices:
- Parse frontmatter with `bytes.SplitN` + line scanning — no YAML library
- Share target resolution between Install and Remove via `resolveTargets`
- Path resolution is 3 unexported helpers, not a separate file

**Files:**
- Create: `instill.go`

**Step 1: Create instill.go**

```go
package instill

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Agent represents a detected AI coding agent.
type Agent struct {
	Name        string
	DisplayName string
	ProjectDir  string // project-level skills dir (relative)
	GlobalDir   string // global skills dir (absolute)
}

// Options configures Install and Remove.
type Options struct {
	Agents     []string // required: agent names to target
	ProjectDir string   // project root (for project-level operations)
	Global     bool     // operate on global dirs instead of project-level
}

// Result reports what happened for each agent.
type Result struct {
	Agent   string
	Path    string
	Created bool // true if newly created (Install) or existed before removal (Remove)
}

// Detect returns agents whose config directories exist in projectDir (or globally).
func Detect(projectDir string, global bool) ([]Agent, error) {
	var out []Agent
	for i := range agents {
		a := &agents[i]
		for _, dd := range a.detectDirs {
			p := resolvePath(dd, projectDir, global)
			if p == "" {
				continue
			}
			if _, err := os.Stat(p); err == nil {
				out = append(out, Agent{a.name, a.displayName, a.skillsDir, resolvePath(a.globalDir, "", true)})
				break
			}
		}
	}
	return out, nil
}

// Install writes skill files from fsys to each target agent's skills directory.
// Skill name is parsed from SKILL.md frontmatter.
func Install(fsys fs.FS, opts Options) ([]Result, error) {
	if len(opts.Agents) == 0 {
		return nil, fmt.Errorf("instill: no agents specified")
	}
	skills, err := findSkills(fsys)
	if err != nil {
		return nil, err
	}
	if len(skills) == 0 {
		return nil, fmt.Errorf("instill: no SKILL.md found in provided filesystem")
	}
	targets, err := resolveTargets(opts)
	if err != nil {
		return nil, err
	}
	var results []Result
	for _, s := range skills {
		for dir, agentNames := range targets {
			skillDir := filepath.Join(dir, s.name)
			_, err := os.Stat(skillDir)
			created := os.IsNotExist(err)
			if writeErr := writeFiles(skillDir, s.files); writeErr != nil {
				return nil, fmt.Errorf("instill: writing to %s: %w", skillDir, writeErr)
			}
			for _, an := range agentNames {
				results = append(results, Result{an, skillDir, created})
			}
		}
	}
	return results, nil
}

// Remove deletes installed skill files by name.
func Remove(skillName string, opts Options) ([]Result, error) {
	if len(opts.Agents) == 0 {
		return nil, fmt.Errorf("instill: no agents specified")
	}
	if skillName == "" {
		return nil, fmt.Errorf("instill: skill name required")
	}
	targets, err := resolveTargets(opts)
	if err != nil {
		return nil, err
	}
	var results []Result
	for dir, agentNames := range targets {
		skillDir := filepath.Join(dir, skillName)
		_, err := os.Stat(skillDir)
		existed := err == nil
		if existed {
			if rmErr := os.RemoveAll(skillDir); rmErr != nil {
				return nil, fmt.Errorf("instill: removing %s: %w", skillDir, rmErr)
			}
		}
		for _, an := range agentNames {
			results = append(results, Result{an, skillDir, existed})
		}
	}
	return results, nil
}

// --- internal helpers ---

// resolveTargets maps deduplicated resolved dirs to agent names.
func resolveTargets(opts Options) (map[string][]string, error) {
	targets := map[string][]string{}
	for _, name := range opts.Agents {
		a, ok := agentIndex[name]
		if !ok {
			return nil, fmt.Errorf("instill: unknown agent %q", name)
		}
		var dir string
		if opts.Global {
			dir = resolvePath(a.globalDir, "", true)
		} else {
			dir = filepath.Join(opts.ProjectDir, a.skillsDir)
		}
		targets[dir] = append(targets[dir], name)
	}
	return targets, nil
}

type skillEntry struct {
	name  string
	files map[string][]byte // relative path -> content
}

func findSkills(fsys fs.FS) ([]skillEntry, error) {
	var out []skillEntry
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		name, err := parseName(data)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		// Collect all files under this skill's directory
		skillDir := filepath.Dir(p)
		files := map[string][]byte{}
		walkErr := fs.WalkDir(fsys, skillDir, func(fp string, fd fs.DirEntry, ferr error) error {
			if ferr != nil || fd.IsDir() {
				return ferr
			}
			content, err := fs.ReadFile(fsys, fp)
			if err != nil {
				return err
			}
			rel := fp
			if skillDir != "." {
				rel, _ = filepath.Rel(skillDir, fp)
			}
			files[rel] = content
			return nil
		})
		if walkErr != nil {
			return walkErr
		}
		out = append(out, skillEntry{name, files})
		return fs.SkipDir
	})
	return out, err
}

// parseName extracts the "name" field from SKILL.md frontmatter without a YAML library.
// Frontmatter format: --- \n key: value \n --- \n content
func parseName(data []byte) (string, error) {
	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return "", fmt.Errorf("missing frontmatter (must start with ---)")
	}
	parts := bytes.SplitN(data, []byte("---"), 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("malformed frontmatter: missing closing ---")
	}
	for _, line := range bytes.Split(parts[1], []byte("\n")) {
		k, v, ok := bytes.Cut(line, []byte(":"))
		if ok && strings.TrimSpace(string(k)) == "name" {
			name := strings.TrimSpace(string(v))
			if name == "" {
				return "", fmt.Errorf("frontmatter 'name' is empty")
			}
			return name, nil
		}
	}
	return "", fmt.Errorf("frontmatter missing required 'name' field")
}

func writeFiles(dir string, files map[string][]byte) error {
	for rel, content := range files {
		target := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// resolvePath expands ~ and env var placeholders. Relative paths are joined to projectDir.
// If global is true and the path is relative, returns empty (not applicable).
func resolvePath(path, projectDir string, global bool) string {
	path = expandEnv(path)
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	if filepath.IsAbs(path) {
		return path
	}
	if global {
		return ""
	}
	return filepath.Join(projectDir, path)
}

var envDefaults = map[string]string{
	"XDG_CONFIG_HOME":  "~/.config",
	"CLAUDE_CONFIG_DIR": "~/.claude",
	"CODEX_HOME":       "~/.codex",
}

func expandEnv(path string) string {
	for k, def := range envDefaults {
		ph := "$" + k
		if strings.Contains(path, ph) {
			v := strings.TrimSpace(os.Getenv(k))
			if v == "" {
				home, _ := os.UserHomeDir()
				v = strings.Replace(def, "~", home, 1)
			}
			path = strings.ReplaceAll(path, ph, v)
		}
	}
	return path
}
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: exit 0

**Step 3: Commit**

```bash
git add -A && git commit -m "feat: add Detect, Install, Remove — complete public API"
```

---

### Task 3: Tests

One file. Covers all public functions + edge cases.

**Files:**
- Create: `instill_test.go`

**Step 1: Write all tests**

Create `instill_test.go`:

```go
package instill

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func skillFS(name string) fstest.MapFS {
	return fstest.MapFS{
		"SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: " + name + "\n---\n\n# " + name + "\n"),
		},
	}
}

func skillFSWithRef(name string) fstest.MapFS {
	return fstest.MapFS{
		"skills/" + name + "/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: " + name + "\n---\n\n# " + name + "\n"),
		},
		"skills/" + name + "/references/commands.md": &fstest.MapFile{
			Data: []byte("# Commands\n"),
		},
	}
}

// --- AgentNames ---

func TestAgentNames(t *testing.T) {
	names := AgentNames()
	if len(names) != len(agents) {
		t.Fatalf("got %d names, want %d", len(names), len(agents))
	}
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Fatalf("not sorted: %q before %q", names[i-1], names[i])
		}
	}
}

// --- Detect ---

func TestDetect(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755)
	os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755)

	got, err := Detect(tmp, false)
	if err != nil {
		t.Fatal(err)
	}

	found := map[string]bool{}
	for _, a := range got {
		found[a.Name] = true
	}
	if !found["claude-code"] {
		t.Error("expected claude-code")
	}
	if !found["cursor"] {
		t.Error("expected cursor")
	}
	if found["windsurf"] {
		t.Error("unexpected windsurf")
	}
}

func TestDetectGlobal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, ".claude"))
	os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755)

	got, err := Detect("", true)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, a := range got {
		found[a.Name] = true
	}
	if !found["claude-code"] {
		t.Error("expected claude-code via CLAUDE_CONFIG_DIR")
	}
}

func TestDetectEmpty(t *testing.T) {
	got, err := Detect(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

// --- Install ---

func TestInstall(t *testing.T) {
	tmp := t.TempDir()

	results, err := Install(skillFSWithRef("my-tool"), Options{
		Agents:     []string{"claude-code"},
		ProjectDir: tmp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Agent != "claude-code" || !results[0].Created {
		t.Fatalf("unexpected results: %+v", results)
	}
	for _, rel := range []string{"SKILL.md", "references/commands.md"} {
		p := filepath.Join(tmp, ".claude/skills/my-tool", rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

func TestInstallOverwrite(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".claude/skills/x"), 0o755)
	os.WriteFile(filepath.Join(tmp, ".claude/skills/x/SKILL.md"), []byte("old"), 0o644)

	results, err := Install(skillFS("x"), Options{Agents: []string{"claude-code"}, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Created {
		t.Error("expected Created=false on overwrite")
	}
	data, _ := os.ReadFile(filepath.Join(tmp, ".claude/skills/x/SKILL.md"))
	if string(data) == "old" {
		t.Error("not overwritten")
	}
}

func TestInstallGlobal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, ".claude"))

	_, err := Install(skillFS("x"), Options{Agents: []string{"claude-code"}, Global: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".claude/skills/x/SKILL.md")); err != nil {
		t.Errorf("not written globally: %v", err)
	}
}

func TestInstallDedup(t *testing.T) {
	tmp := t.TempDir()

	results, err := Install(skillFS("x"), Options{
		Agents:     []string{"amp", "codex"}, // both use .agents/skills
		ProjectDir: tmp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if _, err := os.Stat(filepath.Join(tmp, ".agents/skills/x/SKILL.md")); err != nil {
		t.Error("missing file")
	}
}

func TestInstallErrors(t *testing.T) {
	if _, err := Install(skillFS("x"), Options{ProjectDir: t.TempDir()}); err == nil {
		t.Error("expected error: no agents")
	}
	if _, err := Install(skillFS("x"), Options{Agents: []string{"nope"}, ProjectDir: t.TempDir()}); err == nil {
		t.Error("expected error: unknown agent")
	}
	empty := fstest.MapFS{}
	if _, err := Install(empty, Options{Agents: []string{"claude-code"}, ProjectDir: t.TempDir()}); err == nil {
		t.Error("expected error: no SKILL.md")
	}
}

// --- Remove ---

func TestRemove(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".claude/skills/x")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("x"), 0o644)

	results, err := Remove("x", Options{Agents: []string{"claude-code"}, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Created {
		t.Fatalf("unexpected: %+v", results)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("dir should be gone")
	}
}

func TestRemoveNotInstalled(t *testing.T) {
	results, err := Remove("x", Options{Agents: []string{"claude-code"}, ProjectDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Created {
		t.Error("expected Created=false when nothing to remove")
	}
}

func TestRemoveErrors(t *testing.T) {
	if _, err := Remove("x", Options{ProjectDir: t.TempDir()}); err == nil {
		t.Error("expected error: no agents")
	}
	if _, err := Remove("", Options{Agents: []string{"claude-code"}, ProjectDir: t.TempDir()}); err == nil {
		t.Error("expected error: empty name")
	}
	if _, err := Remove("x", Options{Agents: []string{"nope"}, ProjectDir: t.TempDir()}); err == nil {
		t.Error("expected error: unknown agent")
	}
}

// --- parseName ---

func TestParseName(t *testing.T) {
	for _, tt := range []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"---\nname: foo\n---\ncontent", "foo", false},
		{"---\nname: foo-bar\ndescription: x\n---\n", "foo-bar", false},
		{"no frontmatter", "", true},
		{"---\ndescription: x\n---\n", "", true},
		{"---\nname: \n---\n", "", true},
	} {
		got, err := parseName([]byte(tt.input))
		if tt.wantErr && err == nil {
			t.Errorf("parseName(%q): expected error", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("parseName(%q): %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("parseName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Round trip ---

func TestRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755)
	os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755)

	detected, _ := Detect(tmp, false)
	names := make([]string, len(detected))
	for i, a := range detected {
		names[i] = a.Name
	}

	_, err := Install(skillFS("roundtrip"), Options{Agents: names, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{".claude/skills/roundtrip/SKILL.md", ".cursor/skills/roundtrip/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(tmp, d)); err != nil {
			t.Errorf("missing: %s", d)
		}
	}

	_, err = Remove("roundtrip", Options{Agents: names, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{".claude/skills/roundtrip", ".cursor/skills/roundtrip"} {
		if _, err := os.Stat(filepath.Join(tmp, d)); !os.IsNotExist(err) {
			t.Errorf("not removed: %s", d)
		}
	}
}
```

**Step 2: Run tests**

Run: `go vet ./... && go test ./... -v`
Expected: all PASS

**Step 3: Commit**

```bash
git add -A && git commit -m "test: full test coverage for Detect, Install, Remove"
```

---

### Task 4: README + verify

**Files:**
- Create: `README.md`

**Step 1: Create README.md**

```markdown
# instill

A Go library that instills knowledge into AI coding agents. Zero dependencies.

Install your SKILL.md files to the right directories for Claude Code, Cursor, Windsurf, and 36 other agents. What Cobra did for shell completion, but for agent skills.

## Usage

```go
import (
    "embed"
    "fmt"
    "log"

    "github.com/JetBrains/instill"
)

//go:embed skills/*
var skills embed.FS

func main() {
    // Detect agents in the current project
    agents, _ := instill.Detect(".", false)

    names := make([]string, len(agents))
    for i, a := range agents {
        names[i] = a.Name
    }

    // Install
    results, err := instill.Install(skills, instill.Options{
        Agents:     names,
        ProjectDir: ".",
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, r := range results {
        fmt.Printf("%s: %s\n", r.Agent, r.Path)
    }
}
```

## API

```go
instill.Detect(projectDir string, global bool) ([]Agent, error)
instill.Install(skills fs.FS, opts Options) ([]Result, error)
instill.Remove(skillName string, opts Options) ([]Result, error)
instill.AgentNames() []string
```

## Agent paths

Sourced from [vercel-labs/skills](https://github.com/vercel-labs/skills/blob/main/src/agents.ts). 39 agents supported.
```

**Step 2: Run final verification**

```bash
go vet ./... && go test ./... -cover
```

Expected: all pass, good coverage

**Step 3: Commit**

```bash
git add -A && git commit -m "docs: add README"
```
