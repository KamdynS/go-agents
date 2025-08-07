# LLM Package

- Providers: OpenAI, Anthropic (submodules)
- Status: Complete for chat/completion + streaming; structured output supported
- Next:
  - Add more providers (Azure OpenAI, Google, local runners)
  - Expand error mapping and usage accounting
  - Pluggable output validators

Publishing:
- Tag root and submodules together; document env config and quickstarts.

### Observability
- Wrap any client with `llm.NewInstrumentedClient(client)`
- Spans: `llm.chat`, `llm.completion`, `llm.stream`
- Labels: `genai.model`, `genai.provider`, `genai.finish_reason`, token usage when available
