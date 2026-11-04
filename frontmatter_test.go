package instill

import (
	"strings"
	"testing"
)

func TestFrontmatterEdgeCases(t *testing.T) {
	f, err := parseFrontmatter([]byte("---\nname: x\ndescription: X.\nmetadata:\n  private: &private \"true\"\n  release: &release \"1.2\"\n  internal: *private\n  version: *release\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !f.internal() || f.version() != "1.2" {
		t.Errorf("aliases: internal=%v, version=%q; want true and 1.2", f.internal(), f.version())
	}

	front, body, ok := cutFrontmatter([]byte("---\nname: x\ndescription: |\n  Before\n  ---\n  After\n---\n\n# Body\n"))
	if !ok || !strings.HasSuffix(front, "  After\n") || body != "# Body\n" {
		t.Errorf("block scalar: front = %q, body = %q", front, body)
	}
}
