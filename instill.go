package instill

import (
	"bytes"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Agent represents a detected AI coding agent
type Agent struct {
	Name        string
	DisplayName string
	ProjectDir  string // project-level skills dir (relative)
	GlobalDir   string // global skills dir (absolute)
}

// Options configure Install and Remove
type Options struct {
	Agents     []string // required: agent names to target
	ProjectDir string   // project root (for project-level operations)
	Global     bool     // operate on global dirs instead of project-level
}

// Result reports what happened for each agent
type Result struct {
	Agent        string
	Path         string
	Existed      bool   // true if the skill was already present before this operation
	PriorVersion string // version from previously installed SKILL.md ("" if new)
}

// Detect returns agents whose config directories exist in projectDir (or globally)
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
		for _, dir := range slices.Sorted(maps.Keys(targets)) {
			agentNames := targets[dir]
			skillDir := filepath.Join(dir, s.name)
			_, statErr := os.Stat(skillDir)
			existed := statErr == nil

			priorVersion := installedVersionAt(skillDir)

			if writeErr := writeFiles(skillDir, s.files); writeErr != nil {
				return nil, fmt.Errorf("instill: writing to %s: %w", skillDir, writeErr)
			}

			for _, an := range agentNames {
				results = append(results, Result{an, skillDir, existed, priorVersion})
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
	skillName = sanitizeName(skillName)
	targets, err := resolveTargets(opts)
	if err != nil {
		return nil, err
	}
	var results []Result
	for _, dir := range slices.Sorted(maps.Keys(targets)) {
		agentNames := targets[dir]
		skillDir := filepath.Join(dir, skillName)
		_, statErr := os.Stat(skillDir)
		existed := statErr == nil
		if existed {
			if rmErr := os.RemoveAll(skillDir); rmErr != nil {
				return nil, fmt.Errorf("instill: removing %s: %w", skillDir, rmErr)
			}
		}
		for _, an := range agentNames {
			results = append(results, Result{an, skillDir, existed, ""})
		}
	}
	return results, nil
}

// InstalledVersion returns the version from an installed skill's SKILL.md frontmatter.
// Returns "" if the skill is not installed or has no version field.
func InstalledVersion(skillName string, opts Options) (string, error) {
	targets, err := resolveTargets(opts)
	if err != nil {
		return "", err
	}
	for _, dir := range slices.Sorted(maps.Keys(targets)) {
		if v := installedVersionAt(filepath.Join(dir, skillName)); v != "" {
			return v, nil
		}
	}
	return "", nil
}

// SkillVersion returns the version from the first SKILL.md found in fsys.
// Returns "" if no version field is present.
func SkillVersion(fsys fs.FS) string {
	var version string
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		version, _ = parseFrontmatterField(data, "version")
		return fs.SkipAll
	})
	return version
}

func installedVersionAt(skillDir string) string {
	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return ""
	}
	v, _ := parseFrontmatterField(data, "version")
	return v
}

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
	files map[string][]byte
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
		skillDir := filepath.Dir(p)
		files := map[string][]byte{}
		walkErr := fs.WalkDir(fsys, skillDir, func(fp string, fd fs.DirEntry, ferr error) error {
			if ferr != nil {
				return ferr
			}
			if fd.IsDir() {
				if fd.Name() == ".git" {
					return fs.SkipDir
				}
				return nil
			}
			if isExcluded(fd.Name()) {
				return nil
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

var unsafeChars = regexp.MustCompile(`[^a-z0-9._-]+`)

func sanitizeName(name string) string {
	s := unsafeChars.ReplaceAllString(strings.ToLower(name), "-")
	s = strings.Trim(s, ".-")
	if s == "" {
		return "unnamed-skill"
	}
	if len(s) > 255 {
		s = s[:255]
	}
	return s
}

func parseName(data []byte) (string, error) {
	name, err := parseFrontmatterField(data, "name")
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", fmt.Errorf("frontmatter missing required 'name' field")
	}
	return sanitizeName(name), nil
}

func parseFrontmatterField(data []byte, field string) (string, error) {
	data = bytes.TrimSpace(data)
	if !bytes.HasPrefix(data, []byte("---")) {
		return "", fmt.Errorf("missing frontmatter (must start with ---)")
	}
	parts := bytes.SplitN(data, []byte("---"), 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("malformed frontmatter: missing closing ---")
	}
	for _, line := range bytes.Split(parts[1], []byte("\n")) {
		k, v, ok := bytes.Cut(line, []byte(":"))
		if ok && strings.TrimSpace(string(k)) == field {
			val := strings.TrimSpace(string(v))
			val = strings.Trim(val, `"'`)
			return val, nil
		}
	}
	return "", nil
}

var excludedFiles = map[string]bool{"README.md": true, "metadata.json": true}

func isExcluded(name string) bool {
	return excludedFiles[name] || strings.HasPrefix(name, "_")
}

func writeFiles(dir string, files map[string][]byte) error {
	// Clean stale files from previous installs
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
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
	"XDG_CONFIG_HOME":   "~/.config",
	"CLAUDE_CONFIG_DIR": "~/.claude",
	"CODEX_HOME":        "~/.codex",
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
