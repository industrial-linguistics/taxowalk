package llm

import "context"

type Option struct {
	Name     string
	FullName string
	ID       string
}

type Prompt struct {
	Description string
	Path        []string
	Options     []Option
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Result struct {
	Choice string
	Usage  Usage
}

type Model interface {
	ChooseOption(ctx context.Context, prompt Prompt) (*Result, error)
}
