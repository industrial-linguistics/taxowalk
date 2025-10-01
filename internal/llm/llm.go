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

type Model interface {
	ChooseOption(ctx context.Context, prompt Prompt) (string, error)
}
