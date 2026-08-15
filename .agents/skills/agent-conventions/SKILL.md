---
name: agent-conventions
description: Add or change a skill or subagent in this repo. The caller invokes it whenever a skill is created or edited under .agents/skills/, a subagent wrapper is added or changed under .claude/agents/ or .codex/agents/, or the symlink wiring moves. Covers the one-copy layout, the symlinks, both wrapper formats, and the agents_test.go checks that enforce it.
---

# Cross-agent skills and subagents

Skills and subagents in this repo work in both Claude Code and Codex. Each
skill exists once; the two agents reach it through symlinks. Follow this
whenever you add or change one — `agents_test.go` fails the build if you don't.

## Layout

```
.agents/skills/<skill-name>/SKILL.md        the one real copy
.agents/skills/<skill-name>/references/     supporting material, optional
.claude/skills/<skill-name>   ->  ../../.agents/skills/<skill-name>
.codex/skills/<skill-name>    ->  ../../.agents/skills/<skill-name>
.claude/agents/<agent-name>.md              Claude Code wrapper
.codex/agents/<agent-name>.toml             Codex wrapper
```

## Adding one

1. Write `.agents/skills/<skill-name>/SKILL.md`. Its frontmatter `name:` must
   match the directory name, and `description:` should say when to use it —
   that line is all an agent sees when deciding whether to load the skill.
   Put anything long in `references/` and link to it, so the skill itself stays
   short enough to be read every time.
2. Symlink it into both agent directories, relative so the tree stays portable:

   ```bash
   ln -s ../../.agents/skills/<skill-name> .claude/skills/<skill-name>
   ln -s ../../.agents/skills/<skill-name> .codex/skills/<skill-name>
   ```

3. If a subagent drives the skill, write both wrappers. Keep them thin: the
   skill holds the process, and the wrapper only says who the agent is, what to
   review, and the rules of engagement. Never let the two wrappers drift into
   different instructions — the whole point is that both agents behave the same.
   - `.claude/agents/<agent-name>.md` — YAML frontmatter (`name`, `description`,
     `tools`, `model: inherit`), then the instructions as the body. Write
     `description` so it says what the agent is for and when it applies; Claude
     Code uses it to decide when to delegate.
   - `.codex/agents/<agent-name>.toml` — the same content as `name`,
     `description`, `sandbox_mode` and a `developer_instructions = """…"""`
     block. Use `read-only` for a reviewer that only reads files, and
     `workspace-write` for one that has to build the site or drive a browser.
4. Add the skill to the skills list in `AGENTS.md`.
5. Run `go test ./...`. `agents_test.go` checks the frontmatter name, that both
   symlinks exist and resolve to the shared copy, that every subagent exists for
   both agents, and that each wrapper names the skill it runs.

## Reviewers report; they don't fix

A subagent proposes changes and the caller applies them — so the caller stays
responsible for what lands, and a wrong finding gets argued with rather than
silently committed.
