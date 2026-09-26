// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentTemplatesEmbedded(t *testing.T) {
	files, err := listAgentTemplateFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 8 {
		t.Fatalf("expected embedded templates, got %v", files)
	}
	joined := strings.Join(files, "\n")
	for _, need := range []string{
		"templates/agents/cursor/rules/pisigo-core.mdc",
		"templates/agents/claude/CLAUDE.md",
		"templates/agents/codex/AGENTS.md",
	} {
		if !strings.Contains(joined, need) {
			t.Fatalf("missing embed %s in %v", need, files)
		}
	}
}

func TestInstallAgentsAll(t *testing.T) {
	dir := t.TempDir()
	w, s, err := installAgentTarget(dir, agentTargets["cursor"], false)
	if err != nil || w == 0 {
		t.Fatalf("cursor write=%d skip=%d err=%v", w, s, err)
	}
	w2, s2, err := installAgentTarget(dir, agentTargets["cursor"], false)
	if err != nil || w2 != 0 || s2 == 0 {
		t.Fatalf("expected skip on second install write=%d skip=%d err=%v", w2, s2, err)
	}
	w3, _, err := installAgentTarget(dir, agentTargets["cursor"], true)
	if err != nil || w3 == 0 {
		t.Fatalf("force write=%d err=%v", w3, err)
	}

	for _, name := range []string{"claude", "codex"} {
		if _, _, err := installAgentTarget(dir, agentTargets[name], true); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	checks := []string{
		".cursor/rules/pisigo-core.mdc",
		".cursor/rules/pisigo-structure.mdc",
		".cursor/rules/pisigo-http.mdc",
		".cursor/rules/pisigo-data.mdc",
		"CLAUDE.md",
		".claude/rules/pisigo-http.md",
		".claude/rules/pisigo-data.md",
		".claude/rules/pisigo-structure.md",
		"AGENTS.md",
	}
	for _, rel := range checks {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		body := string(data)
		if !strings.Contains(body, "pisigo.com") || !strings.Contains(strings.ToLower(body), "pisigo") {
			t.Fatalf("%s missing framework guidance", rel)
		}
	}
}

func TestInstallAgentsCLI(t *testing.T) {
	dir := t.TempDir()
	InstallAgents([]string{"all", "-d", dir, "--force"})
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".cursor/rules/pisigo-core.mdc")); err != nil {
		t.Fatal(err)
	}
}
