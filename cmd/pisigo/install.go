package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed templates/agents/**
var agentTemplates embed.FS

type agentTarget struct {
	name    string
	srcRoot string
	mapping []agentFileMap
}

type agentFileMap struct {
	src string // path inside embed FS under templates/agents/<name>/
	dst string // path relative to project root
}

var agentTargets = map[string]agentTarget{
	"cursor": {
		name:    "cursor",
		srcRoot: "templates/agents/cursor",
		mapping: []agentFileMap{
			{src: "rules/pisigo-core.mdc", dst: ".cursor/rules/pisigo-core.mdc"},
			{src: "rules/pisigo-structure.mdc", dst: ".cursor/rules/pisigo-structure.mdc"},
			{src: "rules/pisigo-http.mdc", dst: ".cursor/rules/pisigo-http.mdc"},
			{src: "rules/pisigo-data.mdc", dst: ".cursor/rules/pisigo-data.mdc"},
		},
	},
	"claude": {
		name:    "claude",
		srcRoot: "templates/agents/claude",
		mapping: []agentFileMap{
			{src: "CLAUDE.md", dst: "CLAUDE.md"},
			{src: "rules/pisigo-structure.md", dst: ".claude/rules/pisigo-structure.md"},
			{src: "rules/pisigo-http.md", dst: ".claude/rules/pisigo-http.md"},
			{src: "rules/pisigo-data.md", dst: ".claude/rules/pisigo-data.md"},
		},
	},
	"codex": {
		name:    "codex",
		srcRoot: "templates/agents/codex",
		mapping: []agentFileMap{
			{src: "AGENTS.md", dst: "AGENTS.md"},
		},
	},
}

func Install(args []string) {
	if len(args) < 1 {
		printInstallUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "agents", "agent":
		InstallAgents(args[1:])
	case "help", "-h", "--help":
		printInstallUsage()
	default:
		fatal("unknown install target %q\n\n%s", args[0], installUsageText())
	}
}

func InstallAgents(args []string) {
	force := false
	dir := "."
	var names []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-f" || arg == "--force":
			force = true
		case arg == "-d" || arg == "--dir":
			if i+1 >= len(args) {
				fatal("missing value for %s", arg)
			}
			i++
			dir = args[i]
		case arg == "-h" || arg == "--help" || arg == "help":
			printInstallUsage()
			return
		case strings.HasPrefix(arg, "-"):
			fatal("unknown flag %q", arg)
		default:
			names = append(names, strings.ToLower(arg))
		}
	}

	if len(names) == 0 {
		printInstallUsage()
		os.Exit(1)
	}

	selected := map[string]agentTarget{}
	for _, name := range names {
		if name == "all" {
			for k, v := range agentTargets {
				selected[k] = v
			}
			continue
		}
		t, ok := agentTargets[name]
		if !ok {
			fatal("unknown agent %q (want: cursor, claude, codex, all)", name)
		}
		selected[name] = t
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fatal("%v", err)
	}
	if st, err := os.Stat(absDir); err != nil || !st.IsDir() {
		fatal("target directory does not exist: %s", absDir)
	}

	var written, skipped int
	order := []string{"cursor", "claude", "codex"}
	for _, key := range order {
		t, ok := selected[key]
		if !ok {
			continue
		}
		w, s, err := installAgentTarget(absDir, t, force)
		if err != nil {
			fatal("%v", err)
		}
		written += w
		skipped += s
		fmt.Printf("pisigo: installed %s agent rules → %s\n", t.name, absDir)
	}
	fmt.Printf("pisigo: done (%d written, %d skipped). Use --force to overwrite.\n", written, skipped)
}

func installAgentTarget(root string, t agentTarget, force bool) (written, skipped int, err error) {
	for _, m := range t.mapping {
		srcPath := path.Join(t.srcRoot, m.src)
		data, readErr := agentTemplates.ReadFile(srcPath)
		if readErr != nil {
			return written, skipped, fmt.Errorf("read template %s: %w", srcPath, readErr)
		}
		dst := filepath.Join(root, filepath.FromSlash(m.dst))
		if _, statErr := os.Stat(dst); statErr == nil && !force {
			fmt.Printf("  skip %s (exists)\n", m.dst)
			skipped++
			continue
		}
		if mkErr := os.MkdirAll(filepath.Dir(dst), 0o755); mkErr != nil {
			return written, skipped, mkErr
		}
		if writeErr := os.WriteFile(dst, data, 0o644); writeErr != nil {
			return written, skipped, writeErr
		}
		fmt.Printf("  write %s\n", m.dst)
		written++
	}
	return written, skipped, nil
}

func printInstallUsage() {
	fmt.Print(installUsageText())
}

func installUsageText() string {
	return `pisigo install — install project tooling into the current app

Usage:
  pisigo install agents <target> [targets...] [flags]

Targets:
  cursor    Cursor rules in .cursor/rules/*.mdc
  claude    Claude Code: CLAUDE.md + .claude/rules/*.md
  codex     OpenAI Codex: AGENTS.md
  all       Install all of the above

Flags:
  -d, --dir <path>   project root (default ".")
  -f, --force        overwrite existing files

Examples:
  pisigo install agents all
  pisigo install agents cursor claude
  pisigo install agents codex -d ./myapp --force

Installed files teach agents how Pisigo works (routing, handlers, middleware,
auth, data adapters, project layout) so they can build features with less guesswork.
`
}

func listAgentTemplateFiles() ([]string, error) {
	var out []string
	err := fs.WalkDir(agentTemplates, "templates/agents", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	return out, err
}
