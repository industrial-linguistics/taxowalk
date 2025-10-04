package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIModel struct {
	client *openai.Client
	model  string
}

func NewOpenAIModel(apiKey string, opts ...OptionFunc) (*OpenAIModel, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("openai api key is empty")
	}
	cfg := openai.DefaultConfig(apiKey)
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	client := openai.NewClientWithConfig(cfg)
	return &OpenAIModel{client: client, model: "gpt-5-mini"}, nil
}

type OptionFunc interface {
	apply(cfg *openai.ClientConfig)
}

type optionFunc func(cfg *openai.ClientConfig)

func (f optionFunc) apply(cfg *openai.ClientConfig) {
	f(cfg)
}

func WithBaseURL(url string) OptionFunc {
	return optionFunc(func(cfg *openai.ClientConfig) {
		if strings.TrimSpace(url) != "" {
			cfg.BaseURL = url
		}
	})
}

func (m *OpenAIModel) ChooseOption(ctx context.Context, prompt Prompt) (*Result, error) {
	if m == nil {
		return nil, errors.New("model is nil")
	}
	sb := &strings.Builder{}
	sb.WriteString("You are an expert Shopify taxonomy classifier.\n")
	sb.WriteString("Select the single best matching category from the provided list.\n")
	sb.WriteString("Respond with exactly one of the candidate category names, their full names, the category ID, or the phrase 'none of these'.\n")
	sb.WriteString("Do not add explanations.\n\n")
	sb.WriteString("Product description:\n")
	sb.WriteString(prompt.Description)
	sb.WriteString("\n\n")
	if len(prompt.Path) > 0 {
		sb.WriteString("Current category path: ")
		sb.WriteString(strings.Join(prompt.Path, " > "))
		sb.WriteString("\n")
	}
	sb.WriteString("Candidate categories:\n")
	for i, opt := range prompt.Options {
		label := opt.FullName
		if strings.TrimSpace(label) == "" {
			label = opt.Name
		}
		if strings.TrimSpace(label) == "" {
			label = opt.ID
		}
		if strings.TrimSpace(opt.ID) != "" {
			sb.WriteString(fmt.Sprintf("%d. %s (id: %s)\n", i+1, label, opt.ID))
		} else {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, label))
		}
	}
	sb.WriteString(fmt.Sprintf("%d. none of these\n", len(prompt.Options)+1))
	sb.WriteString("\nRespond with the category name/full name/id or 'none of these'.")

	resp, err := m.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       m.model,
		Temperature: 0,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You classify Shopify products."},
			{Role: openai.ChatMessageRoleUser, Content: sb.String()},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("no completion choices returned")
	}

	result := &Result{
		Choice: strings.TrimSpace(resp.Choices[0].Message.Content),
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	return result, nil
}
