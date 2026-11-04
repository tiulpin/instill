package instill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func skillFS(name string) fstest.MapFS {
	return skillFSVersioned(name, "")
}

func skillFSVersioned(name, version string) fstest.MapFS {
	fm := "---\nname: " + name + "\ndescription: Test skill.\n"
	if version != "" {
		fm += "version: \"" + version + "\"\n"
	}
	fm += "---\n\n# " + name + "\n"
	return fstest.MapFS{
		"SKILL.md": &fstest.MapFile{Data: []byte(fm)},
	}
}

func skillFSWithRef(name string) fstest.MapFS {
	return fstest.MapFS{
		"skills/" + name + "/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: " + name + "\ndescription: Test skill.\n---\n\n# " + name + "\n"),
		},
		"skills/" + name + "/references/commands.md": &fstest.MapFile{
			Data: []byte("# Commands\n"),
		},
	}
}

func clearRuntimeEnvs(t *testing.T) {
	t.Helper()
	for i := range agents {
		for _, env := range agents[i].runtimeEnvs {
			t.Setenv(env, "")
		}
	}
	t.Setenv("AI_AGENT", "")
	t.Setenv("AGENT", "")
	t.Setenv("CLAUDE_CODE_IS_COWORK", "")
	t.Setenv("CURSOR_TRACE_ID", "")
	t.Setenv("CURSOR_EXTENSION_HOST_ROLE", "")
}

func TestDetectRuntime(t *testing.T) {
	clearRuntimeEnvs(t)

	t.Run("no agent", func(t *testing.T) {
		if got := DetectRuntime(); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("AI_AGENT universal var with known agent", func(t *testing.T) {
		t.Setenv("AI_AGENT", "claude-code")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "claude-code" {
			t.Errorf("Name = %q, want %q", got.Name, "claude-code")
		}
		if got.DisplayName != "Claude Code" {
			t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Claude Code")
		}
		if got.EnvVar != "AI_AGENT" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "AI_AGENT")
		}
	})

	t.Run("AI_AGENT universal var with unknown agent", func(t *testing.T) {
		t.Setenv("AI_AGENT", "future-agent")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "future-agent" {
			t.Errorf("Name = %q, want %q", got.Name, "future-agent")
		}
		if got.EnvVar != "AI_AGENT" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "AI_AGENT")
		}
	})

	t.Run("AI_AGENT versioned form canonicalized", func(t *testing.T) {
		t.Setenv("AI_AGENT", "claude-code_2-1-146_agent")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "claude-code" {
			t.Errorf("Name = %q, want %q", got.Name, "claude-code")
		}
		if got.DisplayName != "Claude Code" {
			t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Claude Code")
		}
		if got.EnvVar != "AI_AGENT" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "AI_AGENT")
		}
	})

	t.Run("AI_AGENT versioned form with unknown name falls through", func(t *testing.T) {
		t.Setenv("AI_AGENT", "future-agent_9-9-9_agent")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "future-agent_9-9-9_agent" {
			t.Errorf("Name = %q, want raw value", got.Name)
		}
	})

	t.Run("AGENT var with known agent", func(t *testing.T) {
		t.Setenv("AGENT", "amp")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "amp" {
			t.Errorf("Name = %q, want %q", got.Name, "amp")
		}
		if got.EnvVar != "AGENT" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "AGENT")
		}
	})

	t.Run("AGENT var with unknown value ignored", func(t *testing.T) {
		t.Setenv("AGENT", "something-random")
		if got := DetectRuntime(); got != nil {
			t.Errorf("unknown AGENT value should be ignored, got %+v", got)
		}
	})

	t.Run("CLAUDECODE", func(t *testing.T) {
		t.Setenv("CLAUDECODE", "1")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "claude-code" {
			t.Errorf("Name = %q, want %q", got.Name, "claude-code")
		}
		if got.EnvVar != "CLAUDECODE" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "CLAUDECODE")
		}
	})

	t.Run("CLAUDE_CODE alias", func(t *testing.T) {
		t.Setenv("CLAUDE_CODE", "1")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "claude-code" {
			t.Errorf("Name = %q, want %q", got.Name, "claude-code")
		}
	})

	t.Run("cowork mode", func(t *testing.T) {
		t.Setenv("CLAUDE_CODE_IS_COWORK", "1")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "cowork" {
			t.Errorf("Name = %q, want %q", got.Name, "cowork")
		}
		if got.EnvVar != "CLAUDE_CODE_IS_COWORK" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "CLAUDE_CODE_IS_COWORK")
		}
	})

	t.Run("codex via CODEX_CI", func(t *testing.T) {
		t.Setenv("CODEX_CI", "1")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "codex" {
			t.Errorf("Name = %q, want %q", got.Name, "codex")
		}
		if got.EnvVar != "CODEX_CI" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "CODEX_CI")
		}
	})

	t.Run("codex via CODEX_THREAD_ID", func(t *testing.T) {
		t.Setenv("CODEX_THREAD_ID", "abc123")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "codex" {
			t.Errorf("Name = %q, want %q", got.Name, "codex")
		}
	})

	t.Run("cursor via CURSOR_TRACE_ID", func(t *testing.T) {
		t.Setenv("CURSOR_TRACE_ID", "trace-123")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "cursor" {
			t.Errorf("Name = %q, want %q", got.Name, "cursor")
		}
		if got.EnvVar != "CURSOR_TRACE_ID" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "CURSOR_TRACE_ID")
		}
	})

	t.Run("copilot via COPILOT_MODEL", func(t *testing.T) {
		t.Setenv("COPILOT_MODEL", "gpt-4")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "github-copilot" {
			t.Errorf("Name = %q, want %q", got.Name, "github-copilot")
		}
	})

	t.Run("opencode via OPENCODE_CLIENT", func(t *testing.T) {
		t.Setenv("OPENCODE_CLIENT", "1")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "opencode" {
			t.Errorf("Name = %q, want %q", got.Name, "opencode")
		}
		if got.EnvVar != "OPENCODE_CLIENT" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "OPENCODE_CLIENT")
		}
	})

	t.Run("AI_AGENT takes priority over specific var", func(t *testing.T) {
		t.Setenv("AI_AGENT", "cursor")
		t.Setenv("CLAUDECODE", "1")
		got := DetectRuntime()
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "cursor" {
			t.Errorf("AI_AGENT should take priority, got Name = %q", got.Name)
		}
		if got.EnvVar != "AI_AGENT" {
			t.Errorf("EnvVar = %q, want %q", got.EnvVar, "AI_AGENT")
		}
	})
}

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

func TestDetect(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, ".claude"))
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}

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
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

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
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, ".claude"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("CODEX_HOME", filepath.Join(tmp, ".codex"))
	t.Setenv("APPDATA", filepath.Join(tmp, "AppData"))

	got, err := Detect(tmp, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestInstall(t *testing.T) {
	tmp := t.TempDir()

	results, err := Install(skillFSWithRef("my-tool"), Options{
		Command:    "tool",
		Agents:     []string{"claude-code"},
		ProjectDir: tmp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Agent != "claude-code" || results[0].Existed {
		t.Fatalf("unexpected results: %+v", results)
	}
	dir := filepath.Join(tmp, ".claude/skills/my-tool")
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nname: my-tool\ndescription: Test skill.\n---\n\n# my-tool\n\nThis file is a discovery stub. Run `tool skill guide my-tool`"
	if !strings.HasPrefix(string(data), want) {
		t.Errorf("stub = %q, want prefix %q", data, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "references/commands.md")); err == nil {
		t.Error("references/commands.md should be served by Guide, not installed")
	}
}

func TestInstallOverwrite(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".claude/skills/x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".claude/skills/x/SKILL.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Install(skillFS("x"), Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Existed {
		t.Error("expected Existed=true on overwrite")
	}
	data, _ := os.ReadFile(filepath.Join(tmp, ".claude/skills/x/SKILL.md"))
	if string(data) == "old" {
		t.Error("content should be updated")
	}
}

func TestInstallGlobal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, ".claude"))

	_, err := Install(skillFS("x"), Options{Command: "tool", Agents: []string{"claude-code"}, Global: true})
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
		Command:    "tool",
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
	if _, err := Install(skillFS("x"), Options{Command: "tool", ProjectDir: t.TempDir()}); err == nil {
		t.Error("expected error: no agents")
	}
	if _, err := Install(skillFS("x"), Options{Command: "tool", Agents: []string{"nope"}, ProjectDir: t.TempDir()}); err == nil {
		t.Error("expected error: unknown agent")
	}
	empty := fstest.MapFS{}
	if _, err := Install(empty, Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: t.TempDir()}); err == nil {
		t.Error("expected error: no SKILL.md")
	}
}

func TestRemove(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".claude/skills/x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Remove("x", Options{Agents: []string{"claude-code"}, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Existed {
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
	if results[0].Existed {
		t.Error("expected Existed=false when nothing to remove")
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

func TestParseName(t *testing.T) {
	for _, tt := range []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"---\nname: foo\n---\ncontent", "foo", false},
		{"---\nname: foo-bar\ndescription: x\n---\n", "foo-bar", false},
		{"---\nname: ../../etc/evil\n---\n", "etc-evil", false},      // sanitized
		{"---\nname: My Cool Tool!\n---\n", "my-cool-tool", false},   // sanitized
		{"---\nname: ...leading-dots\n---\n", "leading-dots", false}, // trimmed
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

func TestInstallExcludesFiles(t *testing.T) {
	tmp := t.TempDir()
	fsys := fstest.MapFS{
		"SKILL.md":       &fstest.MapFile{Data: []byte("---\nname: x\ndescription: Test skill.\n---\n")},
		"README.md":      &fstest.MapFile{Data: []byte("repo readme")},
		"metadata.json":  &fstest.MapFile{Data: []byte("{}")},
		"_internal.md":   &fstest.MapFile{Data: []byte("private")},
		"scripts/run.sh": &fstest.MapFile{Data: []byte("keep this")},
	}
	_, err := Install(fsys, Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmp, ".claude/skills/x")
	for _, excluded := range []string{"README.md", "metadata.json", "_internal.md"} {
		if _, err := os.Stat(filepath.Join(dir, excluded)); err == nil {
			t.Errorf("%s should have been excluded", excluded)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts/run.sh")); err != nil {
		t.Error("scripts/run.sh should have been installed")
	}
}

func TestInstallCleansStaleFiles(t *testing.T) {
	tmp := t.TempDir()
	stale := filepath.Join(tmp, ".claude/skills/x/old-file.md")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Install(skillFS("x"), Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale file should have been cleaned")
	}
}

func TestInstallVersion(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: tmp}

	results, err := Install(skillFSVersioned("x", "1.0"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].PriorVersion != "" {
		t.Errorf("expected empty prior version on fresh install, got %q", results[0].PriorVersion)
	}
	if v, err := InstalledVersion("x", opts); err != nil {
		t.Fatal(err)
	} else if v != "1.0" {
		t.Errorf("InstalledVersion = %q, want %q", v, "1.0")
	}

	// Reinstall with new version
	results, err = Install(skillFSVersioned("x", "2.0"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].PriorVersion != "1.0" {
		t.Errorf("PriorVersion = %q, want %q", results[0].PriorVersion, "1.0")
	}
	if v, err := InstalledVersion("x", opts); err != nil {
		t.Fatal(err)
	} else if v != "2.0" {
		t.Errorf("InstalledVersion after update = %q, want %q", v, "2.0")
	}
}

func TestInstallNoVersion(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: tmp}

	_, err := Install(skillFS("x"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if v, err := InstalledVersion("x", opts); err != nil {
		t.Fatal(err)
	} else if v != "" {
		t.Errorf("InstalledVersion should be empty without version field, got %q", v)
	}
}

func TestSkillVersion(t *testing.T) {
	if v := SkillVersion(skillFSVersioned("x", "3.2.1")); v != "3.2.1" {
		t.Errorf("SkillVersion = %q, want %q", v, "3.2.1")
	}
	if v := SkillVersion(skillFS("x")); v != "" {
		t.Errorf("SkillVersion without version = %q, want empty", v)
	}
	// Nested in skills/ dir
	if v := SkillVersion(skillFSWithRef("my-tool")); v != "" {
		t.Errorf("SkillVersion of ref FS = %q, want empty (no version field)", v)
	}
}

func TestRemovePathTraversal(t *testing.T) {
	tmp := t.TempDir()
	// Create a legitimate skill dir
	dir := filepath.Join(tmp, ".claude/skills/etc-evil")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// ../../etc/evil should be sanitized to etc-evil, not traverse paths
	results, err := Remove("../../etc/evil", Options{Agents: []string{"claude-code"}, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Existed {
		t.Error("expected sanitized name to match etc-evil dir")
	}
	// The real path should never have been touched
	if _, err := os.Stat(filepath.Join(tmp, "../../etc/evil")); err == nil {
		t.Error("path traversal should not reach outside project")
	}
}

func TestInstallDeterministicOrder(t *testing.T) {
	tmp := t.TempDir()
	agents := []string{"claude-code", "cursor"}

	// Run multiple times and verify order is stable
	var first []string
	for i := range 10 {
		dir := filepath.Join(tmp, fmt.Sprintf("run%d", i))
		results, err := Install(skillFS("x"), Options{Command: "tool", Agents: agents, ProjectDir: dir})
		if err != nil {
			t.Fatal(err)
		}
		names := make([]string, len(results))
		for j, r := range results {
			names[j] = r.Agent
		}
		if first == nil {
			first = names
		} else {
			for j := range names {
				if names[j] != first[j] {
					t.Fatalf("run %d: order changed: got %v, want %v", i, names, first)
				}
			}
		}
	}
}

func TestInstalledVersionError(t *testing.T) {
	_, err := InstalledVersion("x", Options{Agents: []string{"bogus-agent"}, ProjectDir: t.TempDir()})
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}

func TestRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, ".claude"))
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}

	detected, _ := Detect(tmp, false)
	names := make([]string, len(detected))
	for i, a := range detected {
		names[i] = a.Name
	}

	_, err := Install(skillFS("roundtrip"), Options{Command: "tool", Agents: names, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{".claude/skills/roundtrip/SKILL.md", ".agents/skills/roundtrip/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(tmp, d)); err != nil {
			t.Errorf("missing: %s", d)
		}
	}

	_, err = Remove("roundtrip", Options{Agents: names, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{".claude/skills/roundtrip", ".agents/skills/roundtrip"} {
		if _, err := os.Stat(filepath.Join(tmp, d)); !os.IsNotExist(err) {
			t.Errorf("not removed: %s", d)
		}
	}
}

func TestExpandEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("VIBE_HOME", "")
	t.Setenv("FLATPAK_XDG_CONFIG_HOME", "")

	if got, want := filepath.Clean(expandEnv("$VIBE_HOME/skills")), filepath.Join(tmp, ".vibe", "skills"); got != want {
		t.Errorf("default: got %q, want %q", got, want)
	}
	t.Setenv("VIBE_HOME", "/opt/vibe")
	if got := expandEnv("$VIBE_HOME/skills"); got != "/opt/vibe/skills" {
		t.Errorf("set: got %q", got)
	}
	if got := expandEnv("$FLATPAK_XDG_CONFIG_HOME/zed"); got != "" {
		t.Errorf("unset without default: got %q, want empty", got)
	}
	if got := resolvePath("$FLATPAK_XDG_CONFIG_HOME/zed", tmp, false); got != "" {
		t.Errorf("unresolved path must not fall back to project dir, got %q", got)
	}
}

func TestInstallGlobalCodexSharesAgentsDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	if _, err := Install(skillFS("x"), Options{Command: "tool", Agents: []string{"codex"}, Global: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".agents/skills/x/SKILL.md")); err != nil {
		t.Errorf("codex global skill not in ~/.agents/skills: %v", err)
	}
}

func TestInstallRequiresDescription(t *testing.T) {
	fsys := fstest.MapFS{"SKILL.md": &fstest.MapFile{Data: []byte("---\nname: x\n---\n")}}
	_, err := Install(fsys, Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "description") {
		t.Errorf("want missing description error, got %v", err)
	}
}

func TestInternalSkills(t *testing.T) {
	fsys := fstest.MapFS{
		"skills/public/SKILL.md": &fstest.MapFile{Data: []byte("---\nname: public\ndescription: Public.\n---\n")},
		"skills/wip/SKILL.md":    &fstest.MapFile{Data: []byte("---\nname: wip\ndescription: WIP.\nmetadata:\n  internal: true\n---\n")},
	}
	if got := ListSkills(fsys); len(got) != 1 || got[0].Name != "public" {
		t.Errorf("ListSkills = %+v, want only public", got)
	}

	tmp := t.TempDir()
	opts := Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: tmp}
	if _, err := Install(fsys, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".claude/skills/wip")); err == nil {
		t.Error("internal skill installed without being named")
	}
	opts.Skills = []string{"wip"}
	if _, err := Install(fsys, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".claude/skills/wip/SKILL.md")); err != nil {
		t.Errorf("named internal skill not installed: %v", err)
	}
}

func TestFrontmatterFields(t *testing.T) {
	f, err := parseFrontmatter([]byte("---\nname: x\ndescription: |\n  Multi\n  line.\nmetadata:\n  name: other\n  version: \"1.2\"\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "x" || f.Description != "Multi\nline.\n" || f.version() != "1.2" {
		t.Errorf("got name %q, description %q, version %q", f.Name, f.Description, f.version())
	}
	if f, _ := parseFrontmatter([]byte("---\nname: x\nversion: 2.0\nmetadata:\n  version: 1.2\n---\n")); f.version() != "2.0" {
		t.Errorf("version = %q, want top-level 2.0", f.version())
	}
}

func TestCutFrontmatter(t *testing.T) {
	front, body, ok := cutFrontmatter([]byte("---\nname: tool\ndescription: Handles foo---bar.\n---\n\n# Instructions\n"))
	if !ok || front != "name: tool\ndescription: Handles foo---bar.\n" || body != "# Instructions\n" {
		t.Errorf("front = %q, body = %q, ok = %v", front, body, ok)
	}
	if _, _, ok := cutFrontmatter([]byte("---\nname: tool\n")); ok {
		t.Error("unclosed frontmatter accepted")
	}
}

func TestInternalSkillYAMLForms(t *testing.T) {
	for name, meta := range map[string]string{
		"inline":  "metadata: {internal: \"true\", version: \"1.2\"}\n",
		"comment": "metadata:\n  internal: \"true\" # private\n",
		"bool":    "metadata:\n  internal: true\n",
	} {
		t.Run(name, func(t *testing.T) {
			fsys := fstest.MapFS{"SKILL.md": &fstest.MapFile{Data: []byte("---\nname: x\ndescription: X.\n" + meta + "---\n")}}
			if got := ListSkills(fsys); len(got) != 0 {
				t.Errorf("ListSkills = %+v, want none", got)
			}
			results, err := Install(fsys, Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: t.TempDir()})
			if err != nil || len(results) != 0 {
				t.Errorf("Install = %+v, %v; want nothing installed", results, err)
			}
		})
	}
}

func TestInstallKeepsResources(t *testing.T) {
	fsys := fstest.MapFS{
		"SKILL.md":               &fstest.MapFile{Data: []byte("---\nname: x\ndescription: X.\n---\n")},
		"assets/template.md":     &fstest.MapFile{Data: []byte("# Template\n")},
		"scripts/notes.md":       &fstest.MapFile{Data: []byte("# Script notes\n")},
		"references/commands.md": &fstest.MapFile{Data: []byte("# Commands\n")},
	}
	tmp := t.TempDir()
	if _, err := Install(fsys, Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: tmp}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmp, ".claude/skills/x")
	for _, rel := range []string{"assets/template.md", "scripts/notes.md"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("%s not installed: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "references/commands.md")); err == nil {
		t.Error("references/commands.md installed; Guide serves it")
	}
	if got := strings.Join(Topics(fsys, "x"), ","); got != "references/commands" {
		t.Errorf("Topics = %q, want only references/commands", got)
	}
}

func TestReinstallRemovesDroppedSubagents(t *testing.T) {
	fsys := fstest.MapFS{
		"SKILL.md":          &fstest.MapFile{Data: []byte("---\nname: x\ndescription: X.\n---\n")},
		"_agents/helper.md": &fstest.MapFile{Data: []byte("# Helper\n")},
	}
	opts := Options{Command: "tool", Agents: []string{"claude-code"}, ProjectDir: t.TempDir()}
	if _, err := Install(fsys, opts); err != nil {
		t.Fatal(err)
	}
	delete(fsys, "_agents/helper.md")
	if _, err := Install(fsys, opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opts.ProjectDir, ".claude/agents/helper.md")); err == nil {
		t.Error("subagent dropped from the skill is still installed")
	}
}

func TestInstallFullCopyWithoutCommand(t *testing.T) {
	tmp := t.TempDir()
	if _, err := Install(skillFSWithRef("my-tool"), Options{Agents: []string{"claude-code"}, ProjectDir: tmp}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmp, ".claude/skills/my-tool")
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "discovery stub") {
		t.Error("SKILL.md replaced by a stub without Command")
	}
	if _, err := os.Stat(filepath.Join(dir, "references/commands.md")); err != nil {
		t.Errorf("references/commands.md not copied: %v", err)
	}
}

func TestDetectRuntimeVercelCompat(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
		want string
		via  string
	}{
		{"agent started from Cursor's terminal", map[string]string{"CURSOR_TRACE_ID": "t", "GEMINI_CLI": "1"}, "gemini-cli", "GEMINI_CLI"},
		{"Cursor agent shell", map[string]string{"CURSOR_EXTENSION_HOST_ROLE": "agent-exec"}, "cursor", "CURSOR_EXTENSION_HOST_ROLE"},
		{"other extension host role", map[string]string{"CURSOR_EXTENSION_HOST_ROLE": "user"}, "", ""},
		{"vercel AI_AGENT name", map[string]string{"AI_AGENT": "claude"}, "claude-code", "AI_AGENT"},
		{"AI_AGENT alias with spaces", map[string]string{"AI_AGENT": " github-copilot-cli "}, "github-copilot", "AI_AGENT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearRuntimeEnvs(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			got := DetectRuntime()
			if tc.want == "" {
				if got != nil && got.EnvVar != "fs:/opt/.devin" {
					t.Errorf("got %+v, want nil", got)
				}
				return
			}
			if got == nil || got.Name != tc.want || got.EnvVar != tc.via {
				t.Errorf("got %+v, want %s via %s", got, tc.want, tc.via)
			}
		})
	}
}

func TestInstallRejectsCollisions(t *testing.T) {
	skill := func(name string) *fstest.MapFile {
		return &fstest.MapFile{Data: []byte("---\nname: " + name + "\ndescription: Test skill.\n---\n")}
	}
	for name, fsys := range map[string]fstest.MapFS{
		"skill name": {"a/tool/SKILL.md": skill("tool"), "b/tool/SKILL.md": skill("tool")},
		"subagent": {
			"a/SKILL.md": skill("a"), "a/_agents/helper.md": {Data: []byte("A")},
			"b/SKILL.md": skill("b"), "b/_agents/helper.md": {Data: []byte("B")},
		},
		"command": {
			"a/SKILL.md": skill("a"), "a/_commands/go.md": {Data: []byte("A")},
			"b/SKILL.md": skill("b"), "b/_commands/go.md": {Data: []byte("B")},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			_, err := Install(fsys, Options{Agents: []string{"claude-code"}, ProjectDir: tmp})
			if err == nil || !strings.Contains(err.Error(), "shipped by both") {
				t.Fatalf("want collision error, got %v", err)
			}
			if entries, _ := os.ReadDir(filepath.Join(tmp, ".claude")); len(entries) != 0 {
				t.Errorf("files written before the collision was detected: %v", entries)
			}
		})
	}
}
