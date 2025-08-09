## CLI: agentctl (optional)

> The CLI is optional. You can ship this repo purely as a library. Only add/maintain the CLI if it materially improves developer experience for your target users.

### TL;DR
- **When to keep CLI**: you want a one-command project scaffold, consistent examples, quick demos, and opinionated defaults.
- **When to skip CLI**: you prefer consumers to create their own apps, frameworks, and deployment; you want to keep this repo focused as a library.
- **Current status**: a minimal `agentctl` exists with `init`, `version`, `help`. It generates small starter apps. It can be expanded or removed without affecting the library.

### Purpose of `cmd/`
- **Go convention**: `cmd/<binary-name>` holds entrypoints for executables, separate from reusable library packages.
- In this repo, `cmd/agentctl` is the developer-facing CLI for scaffolding starter projects and showcasing best practices.

### What exists today
- Commands:
  - `agentctl init [project-name] -type basic|rag|multi-agent`
  - `agentctl version`
  - `agentctl help`

```1:39:go-agents/cmd/agentctl/main.go
func printUsage() {
    fmt.Printf("agentctl - CLI for Go AI Agent framework %s\n\n", version)
    fmt.Println("Usage:")
    fmt.Println("  agentctl init [project-name]  Initialize a new agent project")
    fmt.Println("  agentctl version              Show version information")
    fmt.Println("  agentctl help                 Show this help message")
}
```

- Scaffolding uses inline templates for `main.go`, `go.mod`, `README.md`, `Dockerfile`, `.gitignore`.

```73:101:go-agents/cmd/agentctl/init.go
const goModTemplate = `module %s

go 1.21

require (
    github.com/KamdynS/go-agents v0.1.0
    github.com/KamdynS/go-agents/llm/openai v0.1.0
)
`
```

### Design principles
- **Non-interactive by default**: all commands should support full automation via flags; avoid prompts unless `--interactive` is set.
- **Idempotent**: rerunning `init` should either noop or clearly fail with actionable messages.
- **Small surface**: keep top-level commands few; prefer subcommands only when necessary.
- **Library-first**: CLI should not contain business logic; it should call into library packages or generate code that does.

### Project scaffolds (current and planned)
- `basic`: minimal HTTP agent using `server/http`, in-memory memory, and OpenAI client.
- `rag`: basic Retrieval-Augmented template with `tools/http` ready for external fetches.
- `multi-agent`: stub for coordinator + future specialists.

All templates live as constants in `cmd/agentctl/init.go`. Update or replace them as needed.

### Roadmap options (choose based on appetite)
1) Keep as-is (minimal): acceptable for demos and smoke tests.
2) Expand templates to a microservice layout (recommended for production examples):
   - `cmd/server/main.go`
   - `internal/config`, `internal/http`, `internal/services` (agent orchestration)
   - `Dockerfile`, `docker-compose.yml`, health/metrics endpoints
3) Remove CLI entirely: focus the repo as a pure library.

Example future microservice template (high-level):

```text
my-agent/
  cmd/server/main.go
  internal/config/config.go
  internal/http/server.go
  internal/services/agent/service.go
  go.mod
  Dockerfile
  README.md
```

### Versioning & distribution
- Version with repo tags (SemVer). Users can install with:

```bash
go install github.com/KamdynS/go-agents/cmd/agentctl@vX.Y.Z
```

- Optional future: Homebrew tap, GitHub Releases with prebuilt binaries.

### Testing & CI suggestions
- Build: `go build ./cmd/agentctl`
- Smoke: `go run ./cmd/agentctl help`
- E2E: generate a temp project and `go build` it in CI to ensure templates compile.

### If deferring or removing the CLI
- You can safely keep `cmd/agentctl` unadvertised. The library packages remain unaffected.
- If you want to remove it:
  - Delete `cmd/agentctl`.
  - Strip CLI instructions from `README.md`.
  - Keep examples in `examples/` for guidance instead.

### References
- `cmd/agentctl`: entrypoint and scaffolding logic
  - `cmd/agentctl/main.go`
  - `cmd/agentctl/init.go`

> Note: This framework is intentionally server-agnostic (no built-in CORS/auth). Use the CLI templates as starting points, then apply your own policies in your application or gateway.


