package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Skills live once in .agents/skills/ and are symlinked into .claude/skills/
// and .codex/skills/ so Claude Code and Codex read the same copy. Each skill
// that has a driving subagent needs a wrapper in both .claude/agents/ and
// .codex/agents/. The convention is easy to half-apply — add a skill, forget
// one symlink, and one of the two agents silently stops finding it — so it is
// checked here rather than left to a reviewer's memory.
//
// AGENTS.md "Cross-agent skills and subagents" documents the process this
// enforces.

const (
	sharedSkills = ".agents/skills"
	claudeSkills = ".claude/skills"
	codexSkills  = ".codex/skills"
	claudeAgents = ".claude/agents"
	codexAgents  = ".codex/agents"
)

// skillNames lists the directories under .agents/skills.
func skillNames(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(sharedSkills)
	if err != nil {
		t.Fatalf("read %s: %v", sharedSkills, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatalf("no skills found under %s", sharedSkills)
	}
	return names
}

func TestSkillsHaveFrontmatterName(t *testing.T) {
	for _, skill := range skillNames(t) {
		path := filepath.Join(sharedSkills, skill, "SKILL.md")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		want := "name: " + skill
		if !strings.Contains(string(body), want) {
			t.Errorf("%s: frontmatter should declare %q so both agents resolve the skill by directory name", path, want)
		}
	}
}

func TestSkillsSymlinkedIntoBothAgents(t *testing.T) {
	// Resolve the shared root through any symlinks of its own, so the
	// comparison below is between two fully-resolved absolute paths.
	shared, err := filepath.EvalSymlinks(sharedSkills)
	if err != nil {
		t.Fatalf("resolve %s: %v", sharedSkills, err)
	}
	shared, err = filepath.Abs(shared)
	if err != nil {
		t.Fatalf("abs %s: %v", sharedSkills, err)
	}

	for _, skill := range skillNames(t) {
		want := filepath.Join(shared, skill)

		for _, dir := range []string{claudeSkills, codexSkills} {
			link := filepath.Join(dir, skill)

			info, err := os.Lstat(link)
			if err != nil {
				t.Errorf("%s: missing — every skill must be linked into both %s and %s (ln -s ../../%s/%s %s)",
					link, claudeSkills, codexSkills, sharedSkills, skill, link)
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				t.Errorf("%s: should be a symlink to %s, not a copy — one skill, two agents", link, want)
				continue
			}
			got, err := filepath.EvalSymlinks(link)
			if err != nil {
				t.Errorf("%s: dangling symlink: %v", link, err)
				continue
			}
			if got, err = filepath.Abs(got); err != nil {
				t.Errorf("%s: abs: %v", link, err)
				continue
			}
			if got != want {
				t.Errorf("%s: resolves to %s, want %s", link, got, want)
			}
		}
	}
}

func TestSubagentsExistForBothAgents(t *testing.T) {
	claude := agentSet(t, claudeAgents, ".md")
	codex := agentSet(t, codexAgents, ".toml")

	for name := range claude {
		if !codex[name] {
			t.Errorf("%s/%s.md has no counterpart %s/%s.toml — subagents ship for both Claude Code and Codex",
				claudeAgents, name, codexAgents, name)
		}
	}
	for name := range codex {
		if !claude[name] {
			t.Errorf("%s/%s.toml has no counterpart %s/%s.md — subagents ship for both Claude Code and Codex",
				codexAgents, name, claudeAgents, name)
		}
	}
}

// agentSet returns the agent names defined in dir, stripped of ext.
func agentSet(t *testing.T, dir, ext string) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			names[strings.TrimSuffix(e.Name(), ext)] = true
		}
	}
	if len(names) == 0 {
		t.Fatalf("no %s agents found under %s", ext, dir)
	}
	return names
}

// Each wrapper should name the skill it drives, so the indirection stays
// traceable from either agent's side.
func TestSubagentsReferenceTheirSkill(t *testing.T) {
	cases := []struct {
		file, skill string
	}{
		{filepath.Join(claudeAgents, "prose-reviewer.md"), "prose-review"},
		{filepath.Join(codexAgents, "prose-reviewer.toml"), "prose-review"},
		{filepath.Join(claudeAgents, "browser-tester.md"), "browser-test"},
		{filepath.Join(codexAgents, "browser-tester.toml"), "browser-test"},
		{filepath.Join(claudeAgents, "design-reviewer.md"), "design-review"},
		{filepath.Join(codexAgents, "design-reviewer.toml"), "design-review"},
		{filepath.Join(claudeAgents, "perf-auditor.md"), "perf-audit"},
		{filepath.Join(codexAgents, "perf-auditor.toml"), "perf-audit"},
	}
	for _, c := range cases {
		body, err := os.ReadFile(c.file)
		if err != nil {
			t.Errorf("%s: %v", c.file, err)
			continue
		}
		if !strings.Contains(string(body), c.skill) {
			t.Errorf("%s: should reference the %q skill it wraps", c.file, c.skill)
		}
	}
}

// TestEveryWrapperNamesAKnownSkill is the derived counterpart to
// TestSubagentsReferenceTheirSkill above. That one pins the exact pairing for
// the wrappers someone remembered to list; this one covers wrappers that don't
// exist yet, so adding a fifth subagent can't ship with no guard at all — the
// half-applied-convention failure this file exists to catch.
func TestEveryWrapperNamesAKnownSkill(t *testing.T) {
	skills := skillNames(t)

	for _, dir := range []struct{ path, ext string }{
		{claudeAgents, ".md"},
		{codexAgents, ".toml"},
	} {
		for name := range agentSet(t, dir.path, dir.ext) {
			path := filepath.Join(dir.path, name+dir.ext)
			body, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}

			named := false
			for _, skill := range skills {
				if strings.Contains(string(body), skill) {
					named = true
					break
				}
			}
			if !named {
				t.Errorf("%s: names none of the skills under %s (%s) — a wrapper should say which skill it runs",
					path, sharedSkills, strings.Join(skills, ", "))
			}
		}
	}
}

// TestSkillsHaveFrontmatterDescription guards the one line an agent sees when
// deciding whether to load a skill. A skill with no description is effectively
// invisible to both agents, and every other check in this file still passes.
func TestSkillsHaveFrontmatterDescription(t *testing.T) {
	const minDescription = 40

	for _, skill := range skillNames(t) {
		path := filepath.Join(sharedSkills, skill, "SKILL.md")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}

		front, ok := frontmatterBlock(string(body))
		if !ok {
			t.Errorf("%s: should open with a --- frontmatter block", path)
			continue
		}

		desc := ""
		for _, line := range strings.Split(front, "\n") {
			if after, found := strings.CutPrefix(strings.TrimSpace(line), "description:"); found {
				desc = strings.TrimSpace(after)
				break
			}
		}
		if desc == "" {
			t.Errorf("%s: frontmatter has no description — that line is all an agent sees when deciding to load the skill", path)
			continue
		}
		if len(desc) < minDescription {
			t.Errorf("%s: description is %d characters; it should say when to use the skill (at least %d)",
				path, len(desc), minDescription)
		}
	}
}

// TestSkillReferencesResolve checks that the references/*.md files a SKILL.md
// mentions actually exist. Each gate keeps its detail in references/ so the
// skill itself stays short enough to read every time; a rename turns that
// detail into a dead link with nothing to notice.
//
// A skill may cite another skill's reference by name — sports-update points at
// the prose review's ai-tells.md — so a mention resolves if it exists under any
// skill, not just the one doing the mentioning. That still fails on a rename,
// which is the case worth catching.
func TestSkillReferencesResolve(t *testing.T) {
	skills := skillNames(t)

	for _, skill := range skills {
		dir := filepath.Join(sharedSkills, skill)
		body, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			t.Errorf("%s: %v", dir, err)
			continue
		}

		for _, ref := range referencePaths(string(body)) {
			found := false
			for _, owner := range skills {
				if _, err := os.Stat(filepath.Join(sharedSkills, owner, ref)); err == nil {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s/SKILL.md mentions %s, which exists under no skill in %s",
					dir, ref, sharedSkills)
			}
		}
	}
}

// frontmatterBlock returns the contents of a leading --- delimited block.
func frontmatterBlock(body string) (string, bool) {
	if !strings.HasPrefix(body, "---\n") {
		return "", false
	}
	rest := body[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

// referencePaths pulls every references/<name>.md mentioned in a skill body.
func referencePaths(body string) []string {
	var out []string
	seen := map[string]bool{}
	for _, field := range strings.FieldsFunc(body, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == '(' || r == ')' ||
			r == '`' || r == '"' || r == ',' || r == '[' || r == ']'
	}) {
		field = strings.TrimSuffix(field, ".")
		if strings.HasPrefix(field, "references/") && strings.HasSuffix(field, ".md") && !seen[field] {
			seen[field] = true
			out = append(out, field)
		}
	}
	return out
}
