package skillcmd

import (
	"bytes"
	"encoding/json"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func tree(name, version string) fstest.MapFS {
	return fstest.MapFS{
		"skills/" + name + "/SKILL.md":               &fstest.MapFile{Data: []byte("---\nname: " + name + "\ndescription: Use " + name + ".\nversion: \"" + version + "\"\n---\n\n# " + name + "\n\nSee [commands](references/commands.md).\n")},
		"skills/" + name + "/references/commands.md": &fstest.MapFile{Data: []byte("# " + name + " commands\n")},
	}
}

func trees(fss ...fstest.MapFS) []iofs.FS {
	out := make([]iofs.FS, len(fss))
	for i, f := range fss {
		out[i] = f
	}
	return out
}

func run(t *testing.T, opts Options, args ...string) (string, error) {
	t.Helper()
	cmd := New(opts)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func project(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	return dir
}

func TestList(t *testing.T) {
	opts := Options{Command: "tool", Skills: trees(tree("tool", "1.0"), tree("extra", "0.1")), Default: "tool"}
	out, err := run(t, opts, "list")
	if err != nil || !strings.Contains(out, "tool   1.0      yes") {
		t.Errorf("list: %v\n%s", err, out)
	}
	out, err = run(t, opts, "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var skills []Skill
	if err := json.Unmarshal([]byte(out), &skills); err != nil || len(skills) != 2 || !skills[0].Default || skills[1].Name != "extra" {
		t.Errorf("list --json = %+v, %v", skills, err)
	}
}

func TestInstallFullCopy(t *testing.T) {
	dir := project(t)
	out, err := run(t, Options{Command: "tool", Skills: trees(tree("tool", "1.0")), Default: "tool"}, "install", "--agent", "claude-code", "--project")
	if err != nil || !strings.Contains(out, "Installed tool for claude-code (1.0)") {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude/skills/tool/references/commands.md")); err != nil {
		t.Errorf("full copy missing references: %v", err)
	}
}

func TestStubAndGuide(t *testing.T) {
	dir := project(t)
	opts := Options{Command: "tool", Skills: trees(tree("tool", "1.0"), tree("extra", "0.1")), Default: "tool", Stub: true}
	if _, err := run(t, opts, "install", "--all", "--agent", "claude-code", "--project"); err != nil {
		t.Fatal(err)
	}
	stub, _ := os.ReadFile(filepath.Join(dir, ".claude/skills/extra/SKILL.md"))
	if !strings.Contains(string(stub), "Run `tool skill guide extra`") {
		t.Errorf("stub = %q", stub)
	}

	for args, want := range map[string]string{
		"guide":                           "See commands (`tool skill guide tool references/commands`)",
		"guide extra":                     "# extra",
		"guide commands":                  "# tool commands",
		"guide extra references/commands": "# extra commands",
	} {
		out, err := run(t, opts, strings.Fields(args)...)
		if err != nil || !strings.Contains(out, want) {
			t.Errorf("%s: %v, want %q in\n%s", args, err, want, out)
		}
	}
	if _, err := run(t, opts, "guide", "nope", "x"); err == nil || !strings.Contains(err.Error(), `unknown skill "nope"`) {
		t.Errorf("unknown skill: %v", err)
	}
	if _, err := run(t, Options{Command: "tool", Skills: opts.Skills, Default: "tool"}, "guide"); err == nil {
		t.Error("guide available without Stub")
	}
}

func TestUpdateAndRemove(t *testing.T) {
	project(t)
	var events []Event
	opts := Options{Command: "tool", Skills: trees(tree("tool", "1.0")), Default: "tool", Track: func(e Event) { events = append(events, e) }}
	args := []string{"--agent", "claude-code", "--project"}
	if _, err := run(t, opts, append([]string{"install"}, args...)...); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, opts, append([]string{"update"}, args...)...)
	if err != nil || !strings.Contains(out, "Skill tool is up to date (1.0)") {
		t.Errorf("update unchanged: %v\n%s", err, out)
	}
	opts.Skills = trees(tree("tool", "1.1"))
	out, err = run(t, opts, append([]string{"update"}, args...)...)
	if err != nil || !strings.Contains(out, "Updated tool for claude-code (1.0 -> 1.1)") {
		t.Errorf("update: %v\n%s", err, out)
	}
	out, err = run(t, opts, append([]string{"remove"}, args...)...)
	if err != nil || !strings.Contains(out, "Removed tool from claude-code") {
		t.Errorf("remove: %v\n%s", err, out)
	}

	want := []Event{
		{"install", "claude-code", "project", false, true},
		{"update", "claude-code", "project", false, true},
		{"remove", "claude-code", "project", false, true},
	}
	if len(events) != len(want) {
		t.Fatalf("events = %+v", events)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, events[i], want[i])
		}
	}
}

func TestNameErrors(t *testing.T) {
	project(t)
	opts := Options{Command: "tool", Skills: trees(tree("tool", "1.0")), Default: "tool"}
	if _, err := run(t, opts, "install", "--all", "tool", "--agent", "claude-code"); err == nil {
		t.Error("--all with names accepted")
	}
	if _, err := run(t, opts, "install", "nope", "--agent", "claude-code"); err == nil || !strings.Contains(err.Error(), "run 'tool skill list'") {
		t.Errorf("unknown skill: %v", err)
	}
}
