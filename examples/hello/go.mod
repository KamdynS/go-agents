module github.com/KamdynS/go-agents/examples/hello

go 1.21

require (
    github.com/KamdynS/go-agents v0.1.0
    github.com/KamdynS/go-agents/llm/openai v0.1.0
)

replace github.com/KamdynS/go-agents => ../..
replace github.com/KamdynS/go-agents/llm/openai => ../../llm/openai