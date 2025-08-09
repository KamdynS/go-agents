// Package agents provides top-level documentation and a minimal facade for the
// go-agents module. The module is organized as multiple subpackages (e.g.
// `llm`, `agent`, `memory`, `observability`, `server`, and `tools`).
//
// Importers typically depend on the subpackages directly, for example:
//
//	import (
//	  "github.com/KamdynS/go-agents/llm"
//	  "github.com/KamdynS/go-agents/agent/core"
//	  "github.com/KamdynS/go-agents/memory"
//	)
//
// The root package intentionally keeps a small surface area to avoid stuttering
// and to keep subpackages composable.
package agents
