# forge-core

A provider-agnostic Go library for building AI agent loops with pluggable tools, memory, and middleware.

Forge Core handles the **LLM call -> tool execution -> response** cycle. You supply a provider, register tools, and forge runs the loop, including error handling, iteration limits, and conversation memory.

## Install

```bash
go get github.com/katasec/forge-core@v0.4.0
```

## Quick Start

```bash
export XAI_API_KEY=xai-...
```

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/katasec/forge-core"
    "github.com/katasec/forge-core/provider/xai"
)

func main() {
    provider := xai.New(os.Getenv("XAI_API_KEY"), xai.ModelGrok4FastNonReasoning)

    agent, err := forge.NewAgent(forge.Config{
        Provider:     provider,
        SystemPrompt: "You are a helpful assistant. Keep responses brief.",
    })
    if err != nil {
        log.Fatal(err)
    }

    resp, err := agent.Ask(context.Background(), "Hello! What are you?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.LastText())
}
```

xAI also supports built-in web search and citation access:

```go
provider := xai.New(
    os.Getenv("XAI_API_KEY"),
    xai.ModelGrok4FastNonReasoning,
    xai.WithWebSearch(),
)
agent, err := forge.NewAgent(forge.Config{Provider: provider})
if err != nil {
    log.Fatal(err)
}

resp, err := agent.Ask(context.Background(), "What changed in Go recently?")
if err != nil {
    log.Fatal(err)
}

for _, c := range provider.LastCitations() {
    fmt.Printf("[%s] %s\n", c.Title, c.URL)
}
```

Use OpenAI by changing the provider import and constructor:

```go
import "github.com/katasec/forge-core/provider/openai"

provider := openai.New(os.Getenv("OPENAI_API_KEY"), openai.ModelGPT54Nano)
```

The `openai` package uses the OpenAI Responses API, including text and image content.

## Core Concepts

### Provider

The `Provider` interface makes a single LLM call. Forge includes first-class xAI and OpenAI providers, or you can implement your own:

```go
type Provider interface {
    Generate(ctx context.Context, req ProviderRequest) (*ProviderResponse, error)
}
```

### Tools

Define tools with `Func[Input, Output]`. The JSON schema for parameters is derived from the input struct at construction time using [invopop/jsonschema](https://github.com/invopop/jsonschema). String and byte-slice outputs are returned as-is; other outputs are encoded as JSON before being sent back to the model.

```go
import "github.com/katasec/forge-core/tool"

type SearchInput struct {
    Query string `json:"query" jsonschema:"description=Search query"`
    Limit int    `json:"limit" jsonschema:"description=Max results"`
}

type SearchResult struct {
    Query   string   `json:"query"`
    Limit   int      `json:"limit"`
    Results []string `json:"results"`
}

func search(ctx context.Context, in SearchInput) (SearchResult, error) {
    return SearchResult{
        Query:   in.Query,
        Limit:   in.Limit,
        Results: []string{"First result", "Second result"},
    }, nil
}

agent, err := forge.NewAgent(forge.Config{
    Provider: provider,
    Tools: []forge.Tool{
        tool.Func[SearchInput, SearchResult]("search", "Search the database", search),
    },
})
```

Or implement the `Tool` interface directly for full control:

```go
type Tool interface {
    Name() string
    Description() string
    Schema() ToolSchema
    Invoke(ctx context.Context, args json.RawMessage) (string, error)
}
```

### Agent Loop

`Agent.Run` executes this loop:

1. Load conversation history from memory
2. Call the provider with messages + tool definitions
3. If the provider says **stop**, return the response
4. If the provider requests **tool use**, execute tools, feed results back, go to 2
5. If **iteration limit** hit, return with `FinishReasonIterLimit`
6. Save conversation to memory

For the common case, use `Ask`:

```go
resp, err := agent.Ask(ctx, "Hello")
fmt.Println(resp.LastText())
```

For multimodal input, use `AskContent`:

```go
import "github.com/katasec/forge-core/message"

resp, err := agent.AskContent(ctx,
    message.Text("Describe this image."),
    message.ImageURL("https://example.com/cat.png"),
)
fmt.Println(resp.LastText())
```

Use `AskIn` when you want to manage multiple named conversations:

```go
resp, err := agent.AskIn(ctx, "support-ticket-123", "What happened last?")
```

Use `Run` when you need full control over message roles, multiple messages, or advanced conversation wiring.

### Error Policy

Controls what happens when a tool returns an error:

- `ErrorPolicyStop` (default): terminate the loop immediately
- `ErrorPolicyContinue`: feed the error back to the LLM so it can adapt

### Middleware

Intercept provider calls for logging, retries, rate limiting, etc:

```go
logging := forge.Middleware(func(next forge.RunFunc) forge.RunFunc {
    return func(ctx context.Context, req forge.ProviderRequest) (*forge.ProviderResponse, error) {
        log.Printf("calling provider with %d messages", len(req.Messages))
        resp, err := next(ctx, req)
        if err == nil {
            log.Printf("provider returned: %s", resp.FinishReason)
        }
        return resp, err
    }
})

agent, _ := forge.NewAgent(forge.Config{
    Provider:   myProvider,
    Middleware: []forge.Middleware{logging},
})
```

Middleware composes as decorators: given `[A, B, C]`, request flows `A -> B -> C -> provider -> C -> B -> A`.

### Memory

Forge uses in-memory conversation history by default. Repeated `Ask` calls on the same agent continue the same default conversation:

```go
agent, _ := forge.NewAgent(forge.Config{
    Provider: myProvider,
})

resp, _ := agent.Ask(ctx, "My name is Ameer.")
resp, _ = agent.Ask(ctx, "What is my name?")
```

For named conversations:

```go
resp, _ := agent.AskIn(ctx, "conv-1", "Hi")
resp, _ = agent.AskIn(ctx, "conv-1", "What did I just say?")
```

Disable memory explicitly for stateless agents:

```go
agent, _ := forge.NewAgent(forge.Config{
    Provider:      myProvider,
    DisableMemory: true,
})
```

Implement `MemoryStore` for persistent storage (SQLite, Redis, etc.):

```go
type MemoryStore interface {
    Load(ctx context.Context, conversationID string) ([]Message, error)
    Save(ctx context.Context, conversationID string, messages []Message) error
    Clear(ctx context.Context, conversationID string) error
}
```

Or opt into a supplied memory implementation explicitly:

```go
import "github.com/katasec/forge-core/memory/inmem"

agent, _ := forge.NewAgent(forge.Config{
    Provider: myProvider,
    Memory:   inmem.New(),
})
```

## License

MIT
