# Skill Registry - Bey API

## Overview
This file registers all available skills for the Bey API project. The agent should load skills from this registry when working on relevant tasks.

---

## Project-Level Skills (`.agents/skills/`)

These skills are **specific to this project** and override global skills if duplicates exist.

| Skill Name | Path | Trigger Keywords | Description |
|------------|------|------------------|-------------|
| **golang-patterns** | `.agents/skills/golang-patterns/SKILL.md` | go, golang, patterns | Idiomatic Go patterns, best practices |
| **golang-testing** | `.agents/skills/golang-testing/SKILL.md` | test, testing, go test | Go testing patterns (table-driven, subtests, benchmarks) |
| **golang-concurrency-patterns** | `.agents/skills/golang-concurrency-patterns/SKILL.md` | goroutine, channel, concurrent, worker pool | Go concurrency patterns (goroutines, channels, sync) |
| **golang-pro** | `.agents/skills/golang-pro/SKILL.md` | advanced, pro | Advanced Go patterns |

---

## Global Skills (`~/.opencode/skills/`)

These skills are **global** and available for any project. Use when project-level skills don't exist.

### Go & APIs
| Skill Name | Path | Trigger Keywords | Description |
|------------|------|------------------|-------------|
| **golang-patterns** | `~/.opencode/skills/golang-patterns/SKILL.md` | go, golang, patterns | Idiomatic Go patterns |
| **golang-testing** | `~/.opencode/skills/golang-testing/SKILL.md` | test, testing, go test | Go testing patterns |
| **golang-gin-api** | `~/.opencode/skills/golang-gin-api/golang-gin-api/SKILL.md` | gin, api, rest, route, handler | Gin REST API patterns |

### SDD (Spec-Driven Development)
| Skill Name | Path | Trigger Keywords | Description |
|------------|------|------------------|-------------|
| **sdd-init** | `~/.opencode/skills/sdd-init/SKILL.md` | sdd init, openspec init | Initialize SDD structure |
| **sdd-explore** | `~/.opencode/skills/sdd-explore/SKILL.md` | explore, investigate, research | Explore/investigate ideas |
| **sdd-propose** | `~/.opencode/skills/sdd-propose/SKILL.md` | propose, proposal | Create change proposal |
| **sdd-spec** | `~/.opencode/skills/sdd-spec/SKILL.md` | spec, specification | Write specifications |
| **sdd-design** | `~/.opencode/skills/sdd-design/SKILL.md` | design, architecture | Technical design |
| **sdd-tasks** | `~/.opencode/skills/sdd-tasks/SKILL.md` | tasks, breakdown | Task breakdown |
| **sdd-apply** | `~/.opencode/skills/sdd-apply/SKILL.md` | implement, apply | Implement tasks |
| **sdd-verify** | `~/.opencode/skills/sdd-verify/SKILL.md` | verify, test, validate | Verify implementation |
| **sdd-archive** | `~/.opencode/skills/sdd-archive/SKILL.md` | archive, sync | Archive completed changes |

### Utilities
| Skill Name | Path | Trigger Keywords | Description |
|------------|------|------------------|-------------|
| **skill-creator** | `~/.opencode/skills/skill-creator/SKILL.md` | create skill, new skill | Create new AI skills |
| **github-pr** | `~/.opencode/skills/github-pr/SKILL.md` | pr, pull request | Create pull requests |
| **jira-task** | `~/.opencode/skills/jira-task/SKILL.md` | jira, ticket | Create Jira tasks |
| **jira-epic** | `~/.opencode/skills/jira-epic/SKILL.md` | epic, jira epic | Create Jira epics |

---

## Skill Loading Priority

When working on a task, follow this order:

1. **Check project-level** (`.agents/skills/`) - Load if matching
2. **Check global** (`~/.opencode/skills/`) - Load if project-level doesn't exist
3. **Apply all patterns** from the loaded skill before writing code

---

## Usage Examples

### Go Testing
```
Task: Write tests for product repository
→ Load: golang-testing (from .agents/skills/)
→ Apply: table-driven tests, subtests, benchmarks
```

### Gin API Development
```
Task: Create new endpoint
→ Load: golang-gin-api (from ~/.opencode/skills/)
→ Apply: handler patterns, routing, middleware
```

### SDD Workflow
```
Task: Implement new feature with specs
→ Load: sdd-explore → sdd-propose → sdd-spec → sdd-design → sdd-tasks → sdd-apply → sdd-verify
→ Apply: full SDD lifecycle
```

---

## Adding New Skills

To add a new skill to this registry:

1. Create the skill file in `.agents/skills/` (project) or `~/.opencode/skills/` (global)
2. Add entry to this registry with:
   - Skill name
   - Path
   - Trigger keywords
   - Description
3. Run: `skill-registry` to update

---

*Last updated: 2026-03-13*
