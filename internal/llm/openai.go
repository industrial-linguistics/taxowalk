package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type OpenAIModel struct {
	client *openai.Client
	model  string
}

const (
	selectionToolName = "select_taxonomy_category"
	noneSelection     = "none_of_these"
)

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
	if len(prompt.Options) == 0 {
		return nil, errors.New("prompt has no options")
	}

	sb := &strings.Builder{}
	sb.WriteString("You are an expert Shopify taxonomy classifier.\n")
	sb.WriteString("Select the single best matching category from the provided list.\n")
	sb.WriteString("Use the provided tool to return exactly one selection.\n")
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
	sb.WriteString("\nUse the tool argument 'selection' with values 1..N to pick a category, or 'none_of_these'.")

	allowedSelections := make([]string, 0, len(prompt.Options)+1)
	for i := range prompt.Options {
		allowedSelections = append(allowedSelections, strconv.Itoa(i+1))
	}
	allowedSelections = append(allowedSelections, noneSelection)

	selectionTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        selectionToolName,
			Description: "Select the best matching taxonomy option.",
			Parameters: jsonschema.Definition{
				Type: jsonschema.Object,
				Properties: map[string]jsonschema.Definition{
					"selection": {
						Type:        jsonschema.String,
						Description: "One-based index for the selected option, or none_of_these.",
						Enum:        allowedSelections,
					},
				},
				Required: []string{"selection"},
			},
		},
	}

	resp, err := m.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       m.model,
		Temperature: 0,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You classify Shopify products."},
			{Role: openai.ChatMessageRoleUser, Content: sb.String()},
		},
		Tools: []openai.Tool{selectionTool},
		ToolChoice: openai.ToolChoice{
			Type: openai.ToolTypeFunction,
			Function: openai.ToolFunction{
				Name: selectionToolName,
			},
		},
		ParallelToolCalls: false,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("no completion choices returned")
	}

	selection, err := parseSelection(resp.Choices[0].Message)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Choice: "none of these",
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	if selection != noneSelection {
		oneBased, err := strconv.Atoi(selection)
		if err != nil {
			return nil, fmt.Errorf("invalid selection value %q from model", selection)
		}
		idx := oneBased - 1
		if idx < 0 || idx >= len(prompt.Options) {
			return nil, fmt.Errorf("selection index %d out of range for %d options", oneBased, len(prompt.Options))
		}
		result.ChoiceIndex = &idx

		selected := prompt.Options[idx]
		switch {
		case strings.TrimSpace(selected.ID) != "":
			result.Choice = selected.ID
		case strings.TrimSpace(selected.FullName) != "":
			result.Choice = selected.FullName
		default:
			result.Choice = selected.Name
		}
	}

	return result, nil
}

type selectionArgs struct {
	Selection string `json:"selection"`
}

func parseSelection(msg openai.ChatCompletionMessage) (string, error) {
	for _, tc := range msg.ToolCalls {
		if tc.Type != openai.ToolTypeFunction || tc.Function.Name != selectionToolName {
			continue
		}
		selection, err := parseSelectionArgs(tc.Function.Arguments)
		if err != nil {
			return "", err
		}
		return selection, nil
	}
	if msg.FunctionCall != nil && msg.FunctionCall.Name == selectionToolName {
		return parseSelectionArgs(msg.FunctionCall.Arguments)
	}
	return "", errors.New("model did not return selection tool call")
}

func parseSelectionArgs(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("model returned empty selection payload")
	}
	var payload selectionArgs
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", fmt.Errorf("failed to parse selection payload: %w", err)
	}
	selection := strings.TrimSpace(payload.Selection)
	if selection == "" {
		return "", errors.New("selection payload missing selection field")
	}
	return selection, nil
}
