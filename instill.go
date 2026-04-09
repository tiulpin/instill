package instill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
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
	Skills     []string // if set, only install/update these skill names (by frontmatter name)
	ProjectDir string   // project root (for project-level operations)
	Global     bool     // operate on global dirs instead of project-level
}

// Result reports what happened for each agent
type Result struct {
	Agent        string
	Skill        string
	Path         string
	Existed      bool     // true if the skill was already present before this operation
	PriorVersion string   // version from previously installed SKILL.md ("" if new)
	Commands     []string // command files installed (e.g., from _commands/)
	Subagents    []string // subagent files installed (e.g., from _agents/)
}

type RuntimeAgent struct {
	Name        string
	DisplayName string
	EnvVar      string // env var that matched, or "fs:/opt/.devin" for filesystem detection
}

// DetectRuntime returns the AI agent currently executing this process, or nil.
func DetectRuntime() *RuntimeAgent {
	if v := os.Getenv("AI_AGENT"); v != "" {
		if a, ok := agentIndex[v]; ok {
			return &RuntimeAgent{a.name, a.displayName, "AI_AGENT"}
		}
		return &RuntimeAgent{v, v, "AI_AGENT"}
	}

	if v := os.Getenv("AGENT"); v != "" {
		if a, ok := agentIndex[v]; ok {
			return &RuntimeAgent{a.name, a.displayName, "AGENT"}
		}
	}

	if os.Getenv("CLAUDE_CODE_IS_COWORK") != "" {
		return &RuntimeAgent{"cowork", "Claude Code (Cowork)", "CLAUDE_CODE_IS_COWORK"}
	}

	for i := range agents {
		a := &agents[i]
		for _, env := range a.runtimeEnvs {
			if env == "AGENT" {
				continue
			}
			if os.Getenv(env) != "" {
				return &RuntimeAgent{a.name, a.displayName, env}
			}
		}
	}

	if _, err := os.Stat("/opt/.devin"); err == nil {
		return &RuntimeAgent{"devin", "Devin", "fs:/opt/.devin"}
	}

	return nil
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
// Files under _commands/ and _agents/ in the skill are installed as commands and
// subagents for agents that support them (e.g., Claude Code).
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
		if len(opts.Skills) > 0 && !slices.Contains(opts.Skills, s.name) {
			continue
		}
		for _, dir := range slices.Sorted(maps.Keys(targets)) {
			agentNames := targets[dir]
			skillDir := filepath.Join(dir, s.name)
			_, statErr := os.Stat(skillDir)
			existed := statErr == nil

			priorVersion := installedVersionAt(skillDir)

			if writeErr := writeFiles(skillDir, s.files); writeErr != nil {
				return nil, fmt.Errorf("instill: writing to %s: %w", skillDir, writeErr)
			}

			// Write manifest for removal if skill ships commands or subagents
			if len(s.commands) > 0 || len(s.subagents) > 0 {
				writeManifest(skillDir, sortedKeys(s.commands), sortedKeys(s.subagents))
			}

			for _, an := range agentNames {
				r := Result{Agent: an, Skill: s.name, Path: skillDir, Existed: existed, PriorVersion: priorVersion}

				if cmds, installErr := installExtras(s.commands, an, commandsDirs, opts); installErr != nil {
					return nil, installErr
				} else {
					r.Commands = cmds
				}

				if subs, installErr := installExtras(s.subagents, an, subagentsDirs, opts); installErr != nil {
					return nil, installErr
				} else {
					r.Subagents = subs
				}

				results = append(results, r)
			}
		}
	}
	return results, nil
}

// Remove deletes installed skill files by name, including any commands and
// subagents that were installed alongside the skill.
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

		// Read manifest before deleting the skill directory
		m := readManifest(skillDir)

		if existed {
			if rmErr := os.RemoveAll(skillDir); rmErr != nil {
				return nil, fmt.Errorf("instill: removing %s: %w", skillDir, rmErr)
			}
		}
		for _, an := range agentNames {
			removeExtras(m.Commands, an, commandsDirs, opts)
			removeExtras(m.Subagents, an, subagentsDirs, opts)
			results = append(results, Result{Agent: an, Skill: skillName, Path: skillDir, Existed: existed})
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

// SkillMeta holds metadata parsed from a SKILL.md frontmatter.
type SkillMeta struct {
	Name        string
	Version     string
	Description string
}

// ListSkills returns metadata for every skill found in fsys.
func ListSkills(fsys fs.FS) []SkillMeta {
	var out []SkillMeta
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		name, _ := parseFrontmatterField(data, "name")
		if name == "" {
			return fs.SkipDir
		}
		version, _ := parseFrontmatterField(data, "version")
		desc, _ := parseFrontmatterField(data, "description")
		out = append(out, SkillMeta{Name: sanitizeName(name), Version: version, Description: desc})
		return fs.SkipDir
	})
	return out
}

// SkillVersion returns the version from the first SKILL.md found in fsys.
// Returns "" if no version field is present.
func SkillVersion(fsys fs.FS) string {
	if skills := ListSkills(fsys); len(skills) > 0 {
		return skills[0].Version
	}
	return ""
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
	name      string
	files     map[string][]byte // regular skill files
	commands  map[string][]byte // files from _commands/ (filename → content)
	subagents map[string][]byte // files from _agents/ (filename → content)
}

// skipDirs are directories excluded from regular skill file collection.
// Their contents are handled separately (commands, subagents) or ignored.
var skipDirs = map[string]bool{".git": true, "_commands": true, "_agents": true}

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
		skillDir := path.Dir(p)
		files := map[string][]byte{}
		walkErr := fs.WalkDir(fsys, skillDir, func(fp string, fd fs.DirEntry, ferr error) error {
			if ferr != nil {
				return ferr
			}
			if fd.IsDir() {
				if skipDirs[fd.Name()] {
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
				rel = strings.TrimPrefix(fp, skillDir+"/")
			}
			files[rel] = content
			return nil
		})
		if walkErr != nil {
			return walkErr
		}
		commands := collectSubdir(fsys, skillDir, "_commands")
		subagents := collectSubdir(fsys, skillDir, "_agents")
		out = append(out, skillEntry{name, files, commands, subagents})
		return fs.SkipDir
	})
	return out, err
}

// collectSubdir reads all files from a subdirectory of a skill, returning
// a map of filename → content. Returns nil if the subdirectory doesn't exist.
func collectSubdir(fsys fs.FS, skillDir, subdir string) map[string][]byte {
	dir := subdir
	if skillDir != "." {
		dir = skillDir + "/" + subdir
	}
	result := map[string][]byte{}
	_ = fs.WalkDir(fsys, dir, func(fp string, fd fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipAll
		}
		if fd.IsDir() {
			return nil
		}
		content, readErr := fs.ReadFile(fsys, fp)
		if readErr != nil {
			return readErr
		}
		result[fd.Name()] = content
		return nil
	})
	if len(result) == 0 {
		return nil
	}
	return result
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

// installExtras writes command or subagent files to the appropriate directory
// for agents that support them. Returns the list of installed filenames.
func installExtras(files map[string][]byte, agentName string, dirs map[string][2]string, opts Options) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	d, ok := dirs[agentName]
	if !ok {
		return nil, nil
	}
	var targetDir string
	if opts.Global {
		targetDir = resolvePath(d[1], "", true)
	} else {
		targetDir = filepath.Join(opts.ProjectDir, d[0])
	}
	if targetDir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("instill: creating %s: %w", targetDir, err)
	}
	var installed []string
	for name, content := range files {
		target := filepath.Join(targetDir, name)
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return nil, fmt.Errorf("instill: writing %s: %w", target, err)
		}
		installed = append(installed, name)
	}
	slices.Sort(installed)
	return installed, nil
}

// removeExtras deletes command or subagent files for agents that support them.
func removeExtras(files []string, agentName string, dirs map[string][2]string, opts Options) {
	if len(files) == 0 {
		return
	}
	d, ok := dirs[agentName]
	if !ok {
		return
	}
	var targetDir string
	if opts.Global {
		targetDir = resolvePath(d[1], "", true)
	} else {
		targetDir = filepath.Join(opts.ProjectDir, d[0])
	}
	if targetDir == "" {
		return
	}
	for _, name := range files {
		_ = os.Remove(filepath.Join(targetDir, name))
	}
}

type extrasManifest struct {
	Commands  []string `json:"commands,omitempty"`
	Subagents []string `json:"subagents,omitempty"`
}

func writeManifest(skillDir string, commands, subagents []string) {
	m := extrasManifest{Commands: commands, Subagents: subagents}
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(skillDir, ".instill.json"), data, 0o644)
}

func readManifest(skillDir string) extrasManifest {
	data, err := os.ReadFile(filepath.Join(skillDir, ".instill.json"))
	if err != nil {
		return extrasManifest{}
	}
	var m extrasManifest
	_ = json.Unmarshal(data, &m)
	return m
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
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
