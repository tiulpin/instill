package instill

import (
	"strings"
	"testing"
	"testing/fstest"
)

func guideFS() fstest.MapFS {
	return fstest.MapFS{
		"skills/tool/SKILL.md":                         &fstest.MapFile{Data: []byte("---\nname: tool\ndescription: Use tool.\n---\n\n# Tool\n\nSee [commands](references/commands.md#run) and [site](https://example.com/a.md).\n")},
		"skills/tool/references/commands.md":           &fstest.MapFile{Data: []byte("# Commands\n\nNext: [runs](workflows/runs.md), [overview](references/workflows.md).\n")},
		"skills/tool/references/workflows.md":          &fstest.MapFile{Data: []byte("# Workflows\n")},
		"skills/tool/references/workflows/runs.md":     &fstest.MapFile{Data: []byte("# Runs\n")},
		"skills/tool/references/workflows/commands.md": &fstest.MapFile{Data: []byte("# Run commands\n")},
		"skills/tool/scripts/run.sh":                   &fstest.MapFile{Data: []byte("#!/bin/sh\n")},
		"skills/tool/_agents/helper.md":                &fstest.MapFile{Data: []byte("# Helper\n")},
		"skills/tool/README.md":                        &fstest.MapFile{Data: []byte("# Readme\n")},
		"skills/other/SKILL.md":                        &fstest.MapFile{Data: []byte("---\nname: other\ndescription: Test skill.\n---\n\n# Other\n")},
	}
}

func TestTopics(t *testing.T) {
	got := strings.Join(Topics(guideFS(), "tool"), ",")
	want := "references/commands,references/workflows,references/workflows/commands,references/workflows/runs"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := Topics(guideFS(), "other"); len(got) != 0 {
		t.Errorf("other: got %v", got)
	}
}

func TestGuideOverview(t *testing.T) {
	got, err := Guide(guideFS(), "tool", "tool", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# Tool",
		"See commands (`tool skill guide tool references/commands`)",
		"[site](https://example.com/a.md)",
		"## Topics",
		"- `references/workflows/runs`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "name: tool") {
		t.Error("frontmatter not stripped")
	}
}

func TestGuideTopic(t *testing.T) {
	got, err := Guide(guideFS(), "tool", "tool", "references/commands.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"runs (`tool skill guide tool references/workflows/runs`)",
		"overview (`tool skill guide tool references/workflows`)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "## Topics") {
		t.Error("topics listed on a topic page")
	}
}

func TestGuideStem(t *testing.T) {
	got, err := Guide(guideFS(), "tool", "tool", "runs")
	if err != nil || !strings.Contains(got, "# Runs") {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := Guide(guideFS(), "tool", "tool", "commands"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("want ambiguous error, got %v", err)
	}
}

func TestGuideErrors(t *testing.T) {
	for _, tc := range []struct{ skill, topic, want string }{
		{"missing", "", `unknown skill "missing"`},
		{"tool", "nope", "references/workflows/runs"},
		{"tool", "../other/SKILL", "invalid topic"},
		{"tool", "_agents/helper", "unknown topic"},
		{"tool", "README", "unknown topic"},
	} {
		_, err := Guide(guideFS(), "tool", tc.skill, tc.topic)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("Guide(%q, %q): want error containing %q, got %v", tc.skill, tc.topic, tc.want, err)
		}
	}
}

func TestGuideRootSkill(t *testing.T) {
	fsys := fstest.MapFS{
		"SKILL.md":               &fstest.MapFile{Data: []byte("---\nname: x\ndescription: Test skill.\n---\n\n# x\n")},
		"references/commands.md": &fstest.MapFile{Data: []byte("# Commands\n")},
	}
	got, err := Guide(fsys, "x", "x", "commands")
	if err != nil || got != "# Commands\n" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestGuideBackAndReferenceLinks(t *testing.T) {
	fsys := fstest.MapFS{
		"SKILL.md":               &fstest.MapFile{Data: []byte("---\nname: x\ndescription: X.\n---\n\nRead [commands][cmd] and [readme](README.md).\n\n[cmd]: references/commands.md\n")},
		"references/commands.md": &fstest.MapFile{Data: []byte("Back to [overview](../SKILL.md).\n")},
		"README.md":              &fstest.MapFile{Data: []byte("# Readme\n")},
	}
	got, err := Guide(fsys, "tool", "x", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "[cmd]: `tool skill guide x references/commands`") || !strings.Contains(got, "[readme](README.md)") {
		t.Errorf("overview links:\n%s", got)
	}
	got, err = Guide(fsys, "tool", "x", "commands")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "overview (`tool skill guide x`)") {
		t.Errorf("backlink: %q", got)
	}
}

func TestGuideKeepsCode(t *testing.T) {
	body := "See [run](references/run.md) or `[raw](references/run.md)`.\n\n```markdown\n[Run](references/run.md)\n[cmd]: references/run.md\n```\n\n~~~\n[Run](references/run.md)\n~~~\n"
	fsys := fstest.MapFS{
		"SKILL.md":          &fstest.MapFile{Data: []byte("---\nname: x\ndescription: X.\n---\n\n" + body)},
		"references/run.md": &fstest.MapFile{Data: []byte("# Run\n")},
	}
	got, err := Guide(fsys, "tool", "x", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"See run (`tool skill guide x references/run`)",
		"`[raw](references/run.md)`",
		"```markdown\n[Run](references/run.md)\n[cmd]: references/run.md\n```",
		"~~~\n[Run](references/run.md)\n~~~",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}
