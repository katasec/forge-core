package openai

type request struct {
	Model        string      `json:"model"`
	Input        []inputItem `json:"input"`
	Instructions string      `json:"instructions,omitempty"`
}

type inputItem struct {
	Role    string         `json:"role"`
	Content []contentInput `json:"content"`
}

type contentInput struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type response struct {
	Output []outputItem `json:"output"`
	Usage  usage        `json:"usage"`
}

type outputItem struct {
	Type    string          `json:"type"`
	Role    string          `json:"role,omitempty"`
	Content []contentOutput `json:"content,omitempty"`
}

type contentOutput struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type usage struct {
	InputTokens         int                 `json:"input_tokens"`
	InputTokensDetails  inputTokensDetails  `json:"input_tokens_details"`
	OutputTokens        int                 `json:"output_tokens"`
	OutputTokensDetails outputTokensDetails `json:"output_tokens_details"`
	TotalTokens         int                 `json:"total_tokens"`
}

type inputTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type outputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}
