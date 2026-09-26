# Chat Console

An interactive prompt backed by a kiln agent. Memory keeps the conversation
going across turns, so follow-up questions work.

## Run

```bash
export ANTHROPIC_API_KEY=sk-ant-...
go run .
```

Pick a provider with `-provider`:

```bash
go run . -provider anthropic     # default
go run . -provider openai        # OPENAI_API_KEY
go run . -provider xai           # XAI_API_KEY
go run . -provider xai-search    # XAI_API_KEY, prints citations
```

Type `exit` or `quit` to stop.

## What's in here

- **Provider choice** behind one flag — the agent code is identical for all four
- **`inmem.New()`** as the conversation store, so the agent remembers earlier turns
- **`agent.Ask`** for the common path, `resp.LastText()` for the answer
- **`xai.WithWebSearch()`** plus `provider.LastCitations()` in the `xai-search` mode
