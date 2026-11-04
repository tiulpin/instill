package instill

import (
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
		_, body, _ := cutFrontmatter(data)
		page := rewriteLinks(topics, ".", body, command, skill)
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
	return rewriteLinks(topics, path.Dir(name), string(data), command, skill), nil
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
		if !isPage(rel) || isExcluded(d.Name()) {
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

var (
	mdLink    = regexp.MustCompile(`\[([^\]]*)\]\(([^()\s]+\.md)(#[^()\s]*)?\)`)
	mdLinkDef = regexp.MustCompile(`(?m)^( {0,3}\[[^\]]+\]:)[ \t]*<?([^\s<>]+\.md)(#\S*)?>?[ \t]*$`)
)

// rewriteLinks turns links to pages Guide serves into guide commands. Other links are left as is.
func rewriteLinks(topics []string, pageDir, page, command, skill string) string {
	page = mdLink.ReplaceAllStringFunc(page, func(m string) string {
		sub := mdLink.FindStringSubmatch(m)
		if cmd, ok := guideLink(topics, pageDir, sub[2], command, skill); ok {
			return sub[1] + " (`" + cmd + "`)"
		}
		return m
	})
	return mdLinkDef.ReplaceAllStringFunc(page, func(m string) string {
		sub := mdLinkDef.FindStringSubmatch(m)
		if cmd, ok := guideLink(topics, pageDir, sub[2], command, skill); ok {
			return sub[1] + " `" + cmd + "`"
		}
		return m
	})
}

// guideLink resolves target against the page's directory, then the skill root, and returns
// the command that prints it: the overview for SKILL.md, a topic for any served page.
func guideLink(topics []string, pageDir, target, command, skill string) (string, bool) {
	if strings.Contains(target, ":") || strings.HasPrefix(target, "/") {
		return "", false
	}
	for _, rel := range []string{path.Join(pageDir, target), path.Clean(target)} {
		if rel == "SKILL.md" {
			return command + " skill guide " + skill, true
		}
		if topic := strings.TrimSuffix(rel, ".md"); slices.Contains(topics, topic) {
			return command + " skill guide " + skill + " " + topic, true
		}
	}
	return "", false
}
