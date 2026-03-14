# Skill Registry - Bey API

As your FIRST step before starting any work, identify and load skills relevant to your task from this registry.

## Project-Level Skills (`.agents/skills/`)

These skills are **specific to this project** and override global skills if duplicates exist.

| Trigger | Skill | Path |
|---------|-------|------|
| Go patterns, idiomatic Go | golang-patterns | .agents/skills/golang-patterns/SKILL.md |
| Go testing, table-driven tests | golang-testing | .agents/skills/golang-testing/SKILL.md |
| Go concurrency, goroutines, channels | golang-concurrency-patterns | .agents/skills/golang-concurrency-patterns/SKILL.md |
| Go microservices, gRPC, pprof | golang-pro | .agents/skills/golang-pro/SKILL.md |
| Docker, containerization, multi-stage builds | docker-expert | .agents/skills/docker-expert/SKILL.md |
| Multi-stage Dockerfile | multi-stage-dockerfile | .agents/skills/multi-stage-dockerfile/SKILL.md |
| PayPal payment, Express Checkout | paypal-integration | .agents/skills/paypal-integration/SKILL.md |
| Design patterns, GoF, architecture | design-patterns-expert | .agents/skills/design-patterns-expert/SKILL.md |

## User Skills (`~/.opencode/skills/`)

These skills are **global** and available for any project. Use when project-level skills don't exist.

### Go & APIs
| Trigger | Skill | Path |
|---------|-------|------|
| Go patterns, idiomatic Go | golang-patterns | ~/.opencode/skills/golang-patterns/SKILL.md |
| Go testing, table-driven tests | golang-testing | ~/.opencode/skills/golang-testing/SKILL.md |
| Gin REST API, Go web server | golang-gin-api | ~/.opencode/skills/golang-gin-api/golang-gin-api/SKILL.md |

### SDD (Spec-Driven Development)
| Trigger | Skill | Path |
|---------|-------|------|
| sdd init, openspec init | sdd-init | ~/.opencode/skills/sdd-init/SKILL.md |
| explore, investigate, research | sdd-explore | ~/.opencode/skills/sdd-explore/SKILL.md |
| propose, proposal | sdd-propose | ~/.opencode/skills/sdd-propose/SKILL.md |
| spec, specification | sdd-spec | ~/.opencode/skills/sdd-spec/SKILL.md |
| design, architecture | sdd-design | ~/.opencode/skills/sdd-design/SKILL.md |
| tasks, breakdown | sdd-tasks | ~/.opencode/skills/sdd-tasks/SKILL.md |
| implement, apply | sdd-apply | ~/.opencode/skills/sdd-apply/SKILL.md |
| verify, test, validate | sdd-verify | ~/.opencode/skills/sdd-verify/SKILL.md |
| archive, sync | sdd-archive | ~/.opencode/skills/sdd-archive/SKILL.md |

### Utilities
| Trigger | Skill | Path |
|---------|-------|------|
| create skill, new skill | skill-creator | ~/.opencode/skills/skill-creator/SKILL.md |
| pr, pull request | github-pr | ~/.opencode/skills/github-pr/SKILL.md |
| jira, ticket | jira-task | ~/.opencode/skills/jira-task/SKILL.md |
| epic, jira epic | jira-epic | ~/.opencode/skills/jira-epic/SKILL.md |

### Frontend Frameworks
| Trigger | Skill | Path |
|---------|-------|------|
| React 19, React Compiler | react-19 | ~/.opencode/skills/react-19/SKILL.md |
| Next.js 15, App Router | nextjs-15 | ~/.opencode/skills/nextjs-15/SKILL.md |
| Zustand 5 state management | zustand-5 | ~/.opencode/skills/zustand-5/SKILL.md |
| Tailwind CSS 4 | tailwind-4 | ~/.opencode/skills/tailwind-4/SKILL.md |
| React Native, Expo | react-native | ~/.opencode/skills/react-native/SKILL.md |

### Angular
| Trigger | Skill | Path |
|---------|-------|------|
| Angular standalone, signals | angular-core | ~/.opencode/skills/angular/core/SKILL.md |
| Angular architecture, project structure | angular-architecture | ~/.opencode/skills/angular/architecture/SKILL.md |
| Angular forms, Reactive Forms | angular-forms | ~/.opencode/skills/angular/forms/SKILL.md |
| Angular performance, NgOptimizedImage | angular-performance | ~/.opencode/skills/angular/performance/SKILL.md |

### TypeScript & Validation
| Trigger | Skill | Path |
|---------|-------|------|
| TypeScript strict patterns | typescript | ~/.opencode/skills/typescript/SKILL.md |
| Zod 4 validation | zod-4 | ~/.opencode/skills/zod-4/SKILL.md |

### Backend
| Trigger | Skill | Path |
|---------|-------|------|
| Django REST Framework | django-drf | ~/.opencode/skills/django-drf/SKILL.md |
| Spring Boot 3 | spring-boot-3 | ~/.opencode/skills/spring-boot-3/SKILL.md |
| Java 21, records, virtual threads | java-21 | ~/.opencode/skills/java-21/SKILL.md |
| Hexagonal architecture Java | hexagonal-architecture-layers-java | ~/.opencode/skills/hexagonal-architecture-layers-java/SKILL.md |

### Testing
| Trigger | Skill | Path |
|---------|-------|------|
| Python pytest | pytest | ~/.opencode/skills/pytest/SKILL.md |
| Playwright E2E testing | playwright | ~/.opencode/skills/playwright/SKILL.md |
| Identity service testing, bun:test | identity-testing | ~/.opencode/skills/identity-testing/SKILL.md |
| Astro testing | astro-testing | ~/.opencode/skills/astro-testing/SKILL.md |

### Other
| Trigger | Skill | Path |
|---------|-------|------|
| Vercel AI SDK 5 | ai-sdk-5 | ~/.opencode/skills/ai-sdk-5/SKILL.md |
| Electron desktop apps | electron | ~/.opencode/skills/electron/SKILL.md |
| Elixir Phoenix antipatterns | elixir-antipatterns | ~/.opencode/skills/elixir-antipatterns/SKILL.md |

## Project Conventions

| File | Path | Notes |
|------|------|-------|
| AGENTS.md | AGENTS.md | Index — references files below |
| cmd/api/main.go | cmd/api/main.go | Entry point |
| internal/config/ | internal/config/ | YAML config loading |
| internal/database/ | internal/database/ | DB connection |
| internal/concurrency/ | internal/concurrency/ | Worker pool, task queue |
| internal/modules/ | internal/modules/ | Feature modules |
| internal/shared/ | internal/shared/ | Middleware, response helpers |
| config.yaml | config.yaml | Configuration file |
| openspec/ | openspec/ | SDD specifications |

Read the convention files listed above for project-specific patterns and rules. All referenced paths have been extracted — no need to read index files to discover more.
