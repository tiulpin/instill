package instill

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

type frontmatter struct {
	Name        string               `yaml:"name"`
	Description string               `yaml:"description"`
	Version     string               `yaml:"version"`
	Metadata    map[string]yaml.Node `yaml:"metadata"`
}

// version reads version from the top level or, per the Agent Skills spec, from metadata.
func (f frontmatter) version() string {
	if f.Version != "" {
		return f.Version
	}
	return f.metadata("version")
}

func (f frontmatter) internal() bool {
	return f.metadata("internal") == "true"
}

// metadata returns a metadata scalar, following YAML aliases.
func (f frontmatter) metadata(key string) string {
	n := f.Metadata[key]
	for n.Kind == yaml.AliasNode && n.Alias != nil {
		n = *n.Alias
	}
	return n.Value
}

func parseFrontmatter(data []byte) (frontmatter, error) {
	var f frontmatter
	front, _, ok := cutFrontmatter(data)
	if !ok {
		return f, fmt.Errorf("missing frontmatter: SKILL.md must start and end it with --- lines")
	}
	if err := yaml.Unmarshal([]byte(front), &f); err != nil {
		return f, fmt.Errorf("frontmatter: %w", err)
	}
	return f, nil
}

// cutFrontmatter splits SKILL.md into the YAML between the --- lines and the body after them.
// The closing --- must start the line, so an indented --- inside a block scalar stays YAML.
func cutFrontmatter(data []byte) (string, string, bool) {
	s := strings.TrimLeft(string(data), "\ufeff\r\n")
	first, rest, ok := strings.Cut(s, "\n")
	if !ok || strings.TrimSpace(first) != "---" {
		return "", string(data), false
	}
	for i := 0; i <= len(rest); {
		line, _, more := strings.Cut(rest[i:], "\n")
		if strings.TrimRight(line, " \t\r") == "---" {
			body := ""
			if more {
				body = rest[i+len(line)+1:]
			}
			return rest[:i], strings.TrimLeft(body, "\r\n"), true
		}
		if !more {
			break
		}
		i += len(line) + 1
	}
	return "", string(data), false
}
