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
	return &OpenAIModel{client: client, model: "gpt-5.4-mini"}, nil
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
	sb.WriteString("\nIf none of the categories match, use selection='none_of_these'.")

	allowedSelections := make([]string, 0, len(prompt.Options)+2)
	for i := range prompt.Options {
		allowedSelections = append(allowedSelections, strconv.Itoa(i+1))
	}
	allowedSelections = append(allowedSelections, strconv.Itoa(len(prompt.Options)+1)) // Backwards-compat for earlier prompts numbering "none of these".
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
		return nil, describeCreateChatCompletionError(err)
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("no completion choices returned")
	}

	selection, err := parseSelection(resp.Choices[0].Message)
	if err != nil {
		return nil, err
	}
	selection = normalizeSelection(selection, len(prompt.Options))

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

func normalizeSelection(selection string, optionCount int) string {
	selection = strings.TrimSpace(selection)
	switch strings.ToLower(selection) {
	case noneSelection, "none", "none of these", "none-of-these":
		return noneSelection
	default:
	}

	oneBased, err := strconv.Atoi(selection)
	if err != nil {
		return selection
	}
	if optionCount > 0 && oneBased == optionCount+1 {
		return noneSelection
	}
	return selection
}

func parseSelection(msg openai.ChatCompletionMessage) (string, error) {
	for _, tc := range msg.ToolCalls {
		if tc.Type != openai.ToolTypeFunction || tc.Function.Name != selectionToolName {
			continue
		}
		return parseSelectionArgs(tc.Function.Arguments)
	}
	if msg.FunctionCall != nil && msg.FunctionCall.Name == selectionToolName {
		return parseSelectionArgs(msg.FunctionCall.Arguments)
	}
	return "", errors.New("model did not return selection tool call")
}

func describeCreateChatCompletionError(err error) error {
	var reqErr *openai.RequestError
	if errors.As(err, &reqErr) {
		if reqErr.Err != nil {
			return fmt.Errorf("chat completion request failed before tool parsing: endpoint returned HTTP %d with a non-JSON error response: %v", reqErr.HTTPStatusCode, reqErr.Err)
		}
		return fmt.Errorf("chat completion request failed before tool parsing: endpoint returned HTTP %d with an unreadable error response", reqErr.HTTPStatusCode)
	}

	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		if msg := strings.TrimSpace(apiErr.Message); msg != "" {
			return fmt.Errorf("chat completion request failed: endpoint returned HTTP %d: %s", apiErr.HTTPStatusCode, msg)
		}
		return fmt.Errorf("chat completion request failed: endpoint returned HTTP %d", apiErr.HTTPStatusCode)
	}

	return err
}

func parseSelectionArgs(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("model returned empty selection payload")
	}

	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to parse selection payload: %w", err)
	}
	rawSelection, ok := payload["selection"]
	if !ok {
		return "", errors.New("selection payload missing selection field")
	}

	var selection string
	switch v := rawSelection.(type) {
	case string:
		selection = v
	case json.Number:
		selection = v.String()
	default:
		return "", fmt.Errorf("selection payload has unsupported selection type %T", rawSelection)
	}

	selection = strings.TrimSpace(selection)
	if selection == "" {
		return "", errors.New("selection payload missing selection field")
	}
	return normalizeSelection(selection, 0), nil
}
