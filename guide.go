package instill

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Guide returns a skill page for an agent: the SKILL.md body when topic is "", otherwise the
// matching .md page by path or stem. Relative .md links become `<command> skill guide` calls.
func Guide(fsys fs.FS, command, skill, topic string) (string, error) {
	dir, err := skillDir(fsys, skill)
	if err != nil {
		return "", err
	}
	topics := Topics(fsys, skill)
	if topic == "" {
		data, err := fs.ReadFile(fsys, path.Join(dir, "SKILL.md"))
		if err != nil {
			return "", err
		}
		page := rewriteLinks(fsys, dir, ".", stripFrontmatter(data), command, skill)
		if len(topics) > 0 {
			page = strings.TrimRight(page, "\n") + fmt.Sprintf("\n\n## Topics\n\nRun `%s skill guide %s <topic>`:\n\n", command, skill)
			for _, t := range topics {
				page += "- `" + t + "`\n"
			}
		}
		return page, nil
	}
	name, err := matchTopic(topics, topic)
	if err != nil {
		return "", fmt.Errorf("instill: %w for skill %q; topics: %s", err, skill, strings.Join(topics, ", "))
	}
	data, err := fs.ReadFile(fsys, path.Join(dir, name+".md"))
	if err != nil {
		return "", err
	}
	return rewriteLinks(fsys, dir, path.Dir(name), string(data), command, skill), nil
}

// Topics lists a skill's .md pages other than SKILL.md, as paths without the extension.
func Topics(fsys fs.FS, skill string) []string {
	dir, err := skillDir(fsys, skill)
	if err != nil {
		return nil
	}
	var out []string
	_ = fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != dir && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), "_")) {
				return fs.SkipDir
			}
			return nil
		}
		rel := strings.TrimPrefix(p, dir+"/")
		if dir == "." {
			rel = p
		}
		if path.Ext(rel) != ".md" || rel == "SKILL.md" || isExcluded(d.Name()) {
			return nil
		}
		out = append(out, strings.TrimSuffix(rel, ".md"))
		return nil
	})
	slices.Sort(out)
	return out
}

func skillDir(fsys fs.FS, skill string) (string, error) {
	dir := ""
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || dir != "" || d.IsDir() || d.Name() != "SKILL.md" {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		if name, _ := parseName(data); name == sanitizeName(skill) {
			dir = path.Dir(p)
			return fs.SkipAll
		}
		return fs.SkipDir
	})
	if dir == "" {
		return "", fmt.Errorf("instill: unknown skill %q", skill)
	}
	return dir, nil
}

func matchTopic(topics []string, topic string) (string, error) {
	topic = strings.TrimSuffix(path.Clean(topic), ".md")
	if !fs.ValidPath(topic) {
		return "", fmt.Errorf("invalid topic %q", topic)
	}
	if slices.Contains(topics, topic) {
		return topic, nil
	}
	var matches []string
	for _, t := range topics {
		if strings.HasSuffix(t, "/"+topic) {
			matches = append(matches, t)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("unknown topic %q", topic)
	default:
		return "", fmt.Errorf("ambiguous topic %q (%s)", topic, strings.Join(matches, ", "))
	}
}

var mdLink = regexp.MustCompile(`\[([^\]]*)\]\(([^()\s]+\.md)(#[^()\s]*)?\)`)

// rewriteLinks turns links to skill pages into guide commands, resolving against the page's
// directory first and the skill root second.
func rewriteLinks(fsys fs.FS, dir, pageDir, page, command, skill string) string {
	return mdLink.ReplaceAllStringFunc(page, func(m string) string {
		sub := mdLink.FindStringSubmatch(m)
		target := sub[2]
		if strings.Contains(target, ":") || strings.HasPrefix(target, "/") {
			return m
		}
		for _, rel := range []string{path.Join(pageDir, target), path.Clean(target)} {
			if !fs.ValidPath(rel) {
				continue
			}
			if _, err := fs.Stat(fsys, path.Join(dir, rel)); err == nil {
				return fmt.Sprintf("%s (`%s skill guide %s %s`)", sub[1], command, skill, strings.TrimSuffix(rel, ".md"))
			}
		}
		return m
	})
}

func stripFrontmatter(data []byte) string {
	trimmed := bytes.TrimSpace(data)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return string(data)
	}
	parts := bytes.SplitN(trimmed, []byte("---"), 3)
	if len(parts) < 3 {
		return string(data)
	}
	return strings.TrimLeft(string(parts[2]), "\n")
}
