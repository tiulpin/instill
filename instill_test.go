package instill

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func skillFS(name string) fstest.MapFS {
	return skillFSVersioned(name, "")
}

func skillFSVersioned(name, version string) fstest.MapFS {
	fm := "---\nname: " + name + "\n"
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
			Data: []byte("---\nname: " + name + "\n---\n\n# " + name + "\n"),
		},
		"skills/" + name + "/references/commands.md": &fstest.MapFile{
			Data: []byte("# Commands\n"),
		},
	}
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
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, ".claude"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("CODEX_HOME", filepath.Join(tmp, ".codex"))

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
		Agents:     []string{"claude-code"},
		ProjectDir: tmp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Agent != "claude-code" || results[0].Existed {
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
	if err := os.MkdirAll(filepath.Join(tmp, ".claude/skills/x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".claude/skills/x/SKILL.md"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Install(skillFS("x"), Options{Agents: []string{"claude-code"}, ProjectDir: tmp})
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
		"SKILL.md":      &fstest.MapFile{Data: []byte("---\nname: x\n---\n")},
		"README.md":     &fstest.MapFile{Data: []byte("repo readme")},
		"metadata.json": &fstest.MapFile{Data: []byte("{}")},
		"_internal.md":  &fstest.MapFile{Data: []byte("private")},
		"references.md": &fstest.MapFile{Data: []byte("keep this")},
	}
	_, err := Install(fsys, Options{Agents: []string{"claude-code"}, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmp, ".claude/skills/x")
	for _, excluded := range []string{"README.md", "metadata.json", "_internal.md"} {
		if _, err := os.Stat(filepath.Join(dir, excluded)); err == nil {
			t.Errorf("%s should have been excluded", excluded)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "references.md")); err != nil {
		t.Error("references.md should have been installed")
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

	_, err := Install(skillFS("x"), Options{Agents: []string{"claude-code"}, ProjectDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale file should have been cleaned")
	}
}

func TestInstallVersion(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{Agents: []string{"claude-code"}, ProjectDir: tmp}

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
	opts := Options{Agents: []string{"claude-code"}, ProjectDir: tmp}

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
	for i := 0; i < 10; i++ {
		dir := filepath.Join(tmp, fmt.Sprintf("run%d", i))
		results, err := Install(skillFS("x"), Options{Agents: agents, ProjectDir: dir})
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
