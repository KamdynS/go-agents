# Principles Parity Checklist

This document tracks feature parity with "Principles of Building AI Agents" and the status in this package.

Legend: Have = implemented, Partial = available but incomplete, Missing = not implemented yet, N/A = out of scope for this repo.

## Part I: Prompting a Large Language Model (LLM)
- Model routing: Partial (basic router via `llm.RouterClient`, `llm.StaticPolicy`)
- Structured output: Have (`llm/structured.go`, OpenAI/Anthropic structured helpers)
- Prompt guidance/system prompts: Have (system prompt in `AgentConfig`)

## Part II: Building an Agent
- Agent loop (reasoning + tools): Have (`agent/core/runner.go`)
- Tool calling: Have (`tools.Registry`, tool-call loop)
- Tool design guidelines: Partial (docs + examples; add schema validation)
- Working memory (conversation): Have (`memory.Store`, in-memory implementation)
- Hierarchical memory: Partial (processors added; need semantic recall/topK range)
- Memory processors (TokenLimiter, ToolCallFilter): Have (`agent/core/agent.go`)
- Dynamic agents (runtime model/instructions/tools): Missing (add `ConfigResolver`)
- Agent middleware (guardrails, auth): Partial (hooks exist; basic guardrails implemented)

## Part III: Tools & MCP
- Popular tools (HTTP/browser): Partial (HTTP request tool; more to add)
- MCP client/server: Missing

## Part IV: Graph-based Workflows
- Builder API `.step()/.then()/.branch()/.when()/.merge()`: Have (`workflow/`)
- Suspend/Resume: Partial (suspend signal + state API; resume semantics next)
- Streaming step updates: Have (events via `WithEvents`)

## Part V: RAG
- Chunking/Embedding/Upsert/Index/Query/Rerank/Synthesis: Partial
  - Interfaces exist (`memory.VectorStore` + pgvector adapter)
  - Missing: `rag/` module (chunkers, embedders, reranker, helpers)
- Alternatives to RAG (Agentic RAG, ReAG, Full Context Loading): Missing (helpers)

## Part VI: Multi-Agent Systems
- Agent supervisor (agents-as-tools): Partial (tool wrapper added)
- Control flow policies: Missing
- Workflows as tools: Partial (tool wrapper added)

## Part VII: Evals
- Textual evals (hallucination, faithfulness, relevancy, etc.): Missing
- Tool usage evals: Missing
- Prompt-variation evals: Missing

## Part VIII: Dev & Deployment
- Local dev UI & tracing: Partial (in-memory tracer/metrics; SSE server)
- Deployment notes: N/A (library-first; examples in separate repos)

## Part IX: Everything Else
- Multimodal (image, voice, video): Missing
- Code generation (sandbox, analyzers): Missing

## Cross-cutting: Streaming & Observability
- Streaming token output: Have (OpenAI/Anthropic streaming; agent RunStream; SSE)
- Observability/tracing: Partial (tracer/metrics interfaces; OTel/Prom exporters planned)

---

## Roadmap (incremental)
- Short term (next):
  - RAG module (`rag/`): chunkers, embeddings (OpenAI), upsert/query, optional reranker
  - Workflow Suspend/Resume storage + API polish
  - Guardrails (input sanitizer + simple allow/deny rules)
- Medium term:
  - MCP client/server shims
  - Agent supervisor and agents-as-tools adapter
  - Evals module (textual evals + CLI)
  - OTel/Prom implementations (swap observability backends)
- Later:
  - A2A shim
  - Multimodal interfaces (STT/TTS, image)
  - Code generation helpers (sandbox runner, lint hooks)

## Status Summary
- Have: agent loop, tools, structured output, streaming, SSE server, memory store, processors, workflow builder, basic router, tracing/metrics interfaces, basic supervisor policies, RAG helpers (chunk/embed/index/query)
- Partial: routing policies, memory (semantic recall), observability exporters (Prom text endpoint; OTel tracer shim), workflows (suspend/resume), dev tooling
- Missing (deferred): MCP, A2A, evals, richer multi-agent policies (debate/vote), multimodal, codegen, RAG reranker


