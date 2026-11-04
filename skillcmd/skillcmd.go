// Package skillcmd builds the `skill` command that CLIs embedding skills share: list,
// install, update, remove and, for stub installs, guide.
package skillcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tiulpin/instill"
)

// Options configures the skill command.
type Options struct {
	Command string  // CLI name, e.g. "teamcity"; appears in help, errors and stubs
	Product string  // name used in help text, e.g. "TeamCity"; defaults to Command
	Skills  []fs.FS // bundled skill trees
	Default string  // skill used when no name is given
	Stub    bool    // install stubs and add `skill guide [skill] [topic]`
	Printer Printer // status output; plain lines on stdout when nil
	Track   func(Event)
}

// Printer writes status messages.
type Printer interface {
	Success(cmd *cobra.Command, msg string)
	Info(cmd *cobra.Command, msg string)
}

// Event describes one install, update or remove for one agent, for analytics.
type Event struct {
	Action       string // "install", "update" or "remove"
	Agent        string
	Scope        string // "global" or "project"
	AutoDetected bool   // agents were detected, not passed with --agent
	OK           bool
}

// Skill is a bundled skill as printed by `skill list --json`.
type Skill struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Default     bool   `json:"default"`
}

// New returns the `skill` command.
func New(opts Options) *cobra.Command {
	if opts.Product == "" {
		opts.Product = opts.Command
	}
	if opts.Printer == nil {
		opts.Printer = plainPrinter{}
	}
	c := &skillCmd{Options: opts}
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage AI coding agent skills",
		Long: fmt.Sprintf(`Install, update, or remove %s skills for AI coding agents.

Skills teach AI coding agents (Claude Code, Cursor, and others) how
to drive the %s CLI. Skills can be installed globally for the
current user or locally into a project.`, opts.Product, opts.Command),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(c.list(), c.install(false), c.install(true), c.remove())
	if opts.Stub {
		cmd.AddCommand(c.guide())
	}
	cmd.SilenceUsage = true
	for _, sub := range cmd.Commands() {
		sub.SilenceUsage = true
	}
	return cmd
}

type skillCmd struct{ Options }

type flags struct {
	agents  []string
	all     bool
	project bool
}

func (c *skillCmd) list() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List skills bundled with this release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var skills []Skill
			for _, s := range c.metas() {
				skills = append(skills, Skill{s.Name, s.Version, s.Description, s.Name == c.Default})
			}
			out := cmd.OutOrStdout()
			if asJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(skills)
			}
			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(tw, "NAME\tVERSION\tDEFAULT\tDESCRIPTION")
			for _, s := range skills {
				def := ""
				if s.Default {
					def = "yes"
				}
				_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.Name, s.Version, def, s.Description)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print skills as JSON")
	return cmd
}

func (c *skillCmd) install(update bool) *cobra.Command {
	f := &flags{}
	use, short, action := "install", "Install skills for AI coding agents", "install"
	if update {
		use, short, action = "update", "Update installed skills to this release", "update"
	}
	cmd := &cobra.Command{
		Use:   use + " [skill...]",
		Short: short,
		Long: fmt.Sprintf(`%s %s skills for AI coding agents.

Without a skill name, uses the default skill (%s); --all selects every bundled skill.
Installs globally unless --project is set, and detects agents unless --agent is set.`,
			strings.ToUpper(action[:1])+action[1:], c.Product, c.Default),
		Example: fmt.Sprintf("  %[1]s skill %[2]s\n  %[1]s skill %[2]s --all\n  %[1]s skill %[2]s --agent claude-code --project", c.Command, use),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runInstall(cmd, f, args, action)
		},
	}
	c.addFlags(cmd, f)
	return cmd
}

func (c *skillCmd) remove() *cobra.Command {
	f := &flags{}
	cmd := &cobra.Command{
		Use:     "remove [skill...]",
		Short:   "Remove skills from AI coding agents",
		Example: fmt.Sprintf("  %[1]s skill remove\n  %[1]s skill remove --all --project", c.Command),
		RunE: func(cmd *cobra.Command, args []string) error {
			agents, auto, err := c.agents(f)
			if err != nil {
				return err
			}
			names, err := c.names(f, args)
			if err != nil {
				return err
			}
			for _, name := range names {
				results, err := instill.Remove(name, instill.Options{Agents: agents, ProjectDir: ".", Global: !f.project})
				if err != nil {
					return err
				}
				for _, r := range results {
					c.track("remove", r.Agent, f, auto, r.Existed)
					if r.Existed {
						c.Printer.Success(cmd, fmt.Sprintf("Removed %s from %s", name, r.Agent))
					} else {
						c.Printer.Info(cmd, fmt.Sprintf("Skill %s is not installed for %s", name, r.Agent))
					}
				}
			}
			return nil
		},
	}
	c.addFlags(cmd, f)
	return cmd
}

func (c *skillCmd) guide() *cobra.Command {
	return &cobra.Command{
		Use:   "guide [skill] [topic]",
		Short: "Print a skill's guide for AI coding agents",
		Long: fmt.Sprintf(`Print a skill's guide. Installed skills point agents here, so the guide always
matches this %s release. Without a topic, prints the overview and the list of topics.`, c.Command),
		Example: fmt.Sprintf("  %[1]s skill guide\n  %[1]s skill guide %[2]s\n  %[1]s skill guide %[2]s <topic>", c.Command, c.Default),
		Args:    cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			skill, topic := c.Default, ""
			switch {
			case len(args) == 2:
				skill, topic = args[0], args[1]
			case len(args) == 1 && c.fsFor(args[0]) != nil:
				skill = args[0]
			case len(args) == 1:
				topic = args[0]
			}
			fsys := c.fsFor(skill)
			if fsys == nil {
				return c.unknown(skill)
			}
			page, err := instill.Guide(fsys, c.Command, skill, topic)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), page)
			return err
		},
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return c.skillNames(), cobra.ShellCompDirectiveNoFileComp
			}
			if fsys := c.fsFor(args[0]); fsys != nil && len(args) == 1 {
				return instill.Topics(fsys, args[0]), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
}

func (c *skillCmd) runInstall(cmd *cobra.Command, f *flags, args []string, action string) error {
	agents, auto, err := c.agents(f)
	if err != nil {
		return err
	}
	names, err := c.names(f, args)
	if err != nil {
		return err
	}
	versions := map[string]string{}
	for _, s := range c.metas() {
		versions[s.Name] = s.Version
	}
	opts := instill.Options{Agents: agents, ProjectDir: ".", Global: !f.project}
	if c.Stub {
		opts.Command = c.Command
	}

	if action == "update" && c.upToDate(names, versions, opts) {
		for _, name := range names {
			c.Printer.Info(cmd, fmt.Sprintf("Skill %s is up to date (%s)", name, versions[name]))
		}
		return nil
	}

	for i, fsys := range c.Skills {
		var here []string
		for _, name := range names {
			if c.treeOf(name) == i {
				here = append(here, name)
			}
		}
		if len(here) == 0 {
			continue
		}
		opts.Skills = here
		results, err := instill.Install(fsys, opts)
		if err != nil {
			for _, agent := range agents {
				c.track(action, agent, f, auto, false)
			}
			return err
		}
		for _, r := range results {
			c.track(action, r.Agent, f, auto, true)
			c.Printer.Success(cmd, installMessage(r, versions[r.Skill]))
		}
	}
	return nil
}

func installMessage(r instill.Result, version string) string {
	switch {
	case !r.Existed:
		return fmt.Sprintf("Installed %s for %s (%s)", r.Skill, r.Agent, version)
	case r.PriorVersion == version:
		return fmt.Sprintf("Reinstalled %s for %s (%s)", r.Skill, r.Agent, version)
	case r.PriorVersion == "":
		return fmt.Sprintf("Updated %s for %s (unversioned -> %s)", r.Skill, r.Agent, version)
	default:
		return fmt.Sprintf("Updated %s for %s (%s -> %s)", r.Skill, r.Agent, r.PriorVersion, version)
	}
}

func (c *skillCmd) upToDate(names []string, versions map[string]string, opts instill.Options) bool {
	for _, name := range names {
		installed, err := instill.InstalledVersion(name, opts)
		if err != nil || versions[name] == "" || installed != versions[name] {
			return false
		}
	}
	return true
}

func (c *skillCmd) addFlags(cmd *cobra.Command, f *flags) {
	cmd.Flags().StringSliceVarP(&f.agents, "agent", "a", nil, "Target agent(s); detected when omitted")
	cmd.Flags().BoolVar(&f.all, "all", false, "Select every bundled skill")
	cmd.Flags().BoolVar(&f.project, "project", false, "Use the current project instead of the global skills directory")
	cmd.ValidArgsFunction = func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return c.skillNames(), cobra.ShellCompDirectiveNoFileComp
	}
	_ = cmd.RegisterFlagCompletionFunc("agent", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return instill.AgentNames(), cobra.ShellCompDirectiveNoFileComp
	})
}

// agents returns the --agent values, or the agents detected for the chosen scope.
func (c *skillCmd) agents(f *flags) ([]string, bool, error) {
	if len(f.agents) > 0 {
		return f.agents, false, nil
	}
	detected, err := instill.Detect(".", !f.project)
	if err != nil {
		return nil, true, fmt.Errorf("detecting agents: %w", err)
	}
	if len(detected) == 0 {
		return nil, true, fmt.Errorf("no AI coding agents detected; use --agent to pick one (run '%s skill install --help' for details)", c.Command)
	}
	names := make([]string, len(detected))
	for i, a := range detected {
		names[i] = a.Name
	}
	return names, true, nil
}

func (c *skillCmd) names(f *flags, args []string) ([]string, error) {
	switch {
	case f.all && len(args) > 0:
		return nil, errors.New("cannot use --all with skill names")
	case f.all:
		return c.skillNames(), nil
	case len(args) == 0:
		args = []string{c.Default}
	}
	for _, name := range args {
		if c.fsFor(name) == nil {
			return nil, c.unknown(name)
		}
	}
	return args, nil
}

func (c *skillCmd) unknown(name string) error {
	return fmt.Errorf("unknown skill %q; run '%s skill list' to see available skills", name, c.Command)
}

func (c *skillCmd) metas() []instill.SkillMeta {
	var out []instill.SkillMeta
	for _, fsys := range c.Skills {
		out = append(out, instill.ListSkills(fsys)...)
	}
	return out
}

func (c *skillCmd) skillNames() []string {
	var out []string
	for _, s := range c.metas() {
		out = append(out, s.Name)
	}
	return out
}

// fsFor returns the tree that bundles skill, including internal skills that list hides.
func (c *skillCmd) fsFor(skill string) fs.FS {
	if i := c.treeOf(skill); i >= 0 {
		return c.Skills[i]
	}
	return nil
}

func (c *skillCmd) treeOf(skill string) int {
	for i, fsys := range c.Skills {
		if _, err := instill.Guide(fsys, c.Command, skill, ""); err == nil {
			return i
		}
	}
	return -1
}

func (c *skillCmd) track(action, agent string, f *flags, auto, ok bool) {
	if c.Track == nil {
		return
	}
	scope := "global"
	if f.project {
		scope = "project"
	}
	c.Track(Event{action, agent, scope, auto, ok})
}

type plainPrinter struct{}

func (plainPrinter) Success(cmd *cobra.Command, msg string) {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), msg)
}
func (plainPrinter) Info(cmd *cobra.Command, msg string) { _, _ = fmt.Fprintln(cmd.OutOrStdout(), msg) }
