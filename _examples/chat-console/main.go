// Chat Console is an interactive prompt backed by a kiln agent: configure the
// agent once, then talk to it with Ask. Memory keeps the conversation going
// across turns.
//
// Run: go run . -provider anthropic
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/katasec/kiln"
	"github.com/katasec/kiln/memory/inmem"
	"github.com/katasec/kiln/provider/anthropic"
	"github.com/katasec/kiln/provider/openai"
	"github.com/katasec/kiln/provider/xai"
)

func main() {
	providerFlag := flag.String("provider", "anthropic", "Provider to use: anthropic, openai, xai, or xai-search")
	flag.Parse()

	provider, afterAnswer := buildProvider(*providerFlag)
	agent, err := kiln.NewAgent(kiln.Config{
		Provider:     provider,
		Memory:       inmem.New(),
		SystemPrompt: "You are a concise assistant. Keep answers short and useful.",
	})
	if err != nil {
		log.Fatal(err)
	}

	runConsole(context.Background(), agent, afterAnswer)
}

// buildProvider returns the selected provider plus a hook that runs after each
// answer, used by xai-search to print the citations backing it.
func buildProvider(name string) (kiln.Provider, func()) {
	switch name {
	case "anthropic":
		key := requireEnv("ANTHROPIC_API_KEY")
		return anthropic.New(key, anthropic.ModelClaudeOpus5), func() {}
	case "openai":
		key := requireEnv("OPENAI_API_KEY")
		return openai.New(key, openai.ModelGPT54Nano), func() {}
	case "xai":
		key := requireEnv("XAI_API_KEY")
		return xai.New(key, xai.ModelGrok4FastNonReasoning), func() {}
	case "xai-search":
		key := requireEnv("XAI_API_KEY")
		provider := xai.New(key, xai.ModelGrok4FastNonReasoning, xai.WithWebSearch())
		return provider, func() { printCitations(provider.LastCitations()) }
	default:
		log.Fatalf("unknown provider %q; use anthropic, openai, xai, or xai-search", name)
		return nil, nil
	}
}

func runConsole(ctx context.Context, agent *kiln.Agent, afterAnswer func()) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("kiln chat console")
	fmt.Println("Type exit or quit to stop.")
	fmt.Println()

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if shouldExit(input) {
			break
		}
		if input == "" {
			continue
		}

		resp, err := agent.Ask(ctx, input)
		if err != nil {
			log.Printf("agent error: %v", err)
			continue
		}

		fmt.Printf("Assistant: %s\n", resp.LastText())
		afterAnswer()
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		log.Printf("read input: %v", err)
	}
}

func shouldExit(input string) bool {
	switch strings.ToLower(input) {
	case "exit", "quit":
		return true
	default:
		return false
	}
}

func requireEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("set %s", name)
	}
	return value
}

func printCitations(citations []xai.Citation) {
	if len(citations) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Sources:")
	for i, citation := range citations {
		fmt.Printf("  [%d] %s - %s\n", i+1, citation.Title, citation.URL)
	}
}
