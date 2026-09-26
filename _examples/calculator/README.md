# Calculator

Demonstrates tool use with a mock provider. No API key needed.

## Run

```bash
go run .
```

## Output

```
User: What is 12 + 30?
----------------------------------------
[middleware] Calling provider with 1 messages
[middleware] Provider returned: finish_reason=tool_use
[middleware] Calling provider with 3 messages
[middleware] Provider returned: finish_reason=stop
----------------------------------------
Assistant: The answer is 42!
Finish reason: stop
Tokens: 65 in, 25 out
Conversation: 4 messages
```

## What's in here

Everything is in `main.go`:

- **Tools**: `add` and `multiply` built with `tool.Func[In, Out]`; the JSON schema comes from the input struct
- **Mock provider**: implements `kiln.Provider` directly — asks for the `add` tool on the first call, answers from the result on the second
- **Logging middleware**: prints each provider call and its finish reason
- **Agent setup**: `kiln.Config` wires provider, tools, middleware, and loop policy
- **Agent loop**: the full cycle — provider call, tool execution, provider call, stop

Useful for seeing how the loop works without API credentials.
