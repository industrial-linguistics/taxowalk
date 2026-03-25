package llm

import (
	"errors"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func TestParseSelectionFromToolCall(t *testing.T) {
	msg := openai.ChatCompletionMessage{
		ToolCalls: []openai.ToolCall{
			{
				ID:   "call_1",
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      selectionToolName,
					Arguments: `{"selection":"2"}`,
				},
			},
		},
	}
	got, err := parseSelection(msg)
	if err != nil {
		t.Fatalf("parseSelection returned error: %v", err)
	}
	if got != "2" {
		t.Fatalf("parseSelection returned %q", got)
	}
}

func TestParseSelectionFromDeprecatedFunctionCall(t *testing.T) {
	msg := openai.ChatCompletionMessage{
		FunctionCall: &openai.FunctionCall{
			Name:      selectionToolName,
			Arguments: `{"selection":"none_of_these"}`,
		},
	}
	got, err := parseSelection(msg)
	if err != nil {
		t.Fatalf("parseSelection returned error: %v", err)
	}
	if got != noneSelection {
		t.Fatalf("parseSelection returned %q", got)
	}
}

func TestParseSelectionMissingToolCall(t *testing.T) {
	msg := openai.ChatCompletionMessage{
		ToolCalls: []openai.ToolCall{
			{
				ID:   "call_1",
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      "other_function",
					Arguments: `{"selection":"2"}`,
				},
			},
		},
	}
	_, err := parseSelection(msg)
	if err == nil {
		t.Fatal("expected error when selection tool call is missing")
	}
}

func TestParseSelectionRejectsContentOnly(t *testing.T) {
	msg := openai.ChatCompletionMessage{
		Content: `{"selection":"none_of_these"}`,
	}
	_, err := parseSelection(msg)
	if err == nil {
		t.Fatal("expected error for content-only selection payload")
	}
}

func TestParseSelectionArgsInvalidJSON(t *testing.T) {
	_, err := parseSelectionArgs("{")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseSelectionArgsMissingField(t *testing.T) {
	_, err := parseSelectionArgs(`{"selection":""}`)
	if err == nil {
		t.Fatal("expected error for missing selection field")
	}
}

func TestParseSelectionArgsAcceptsNumericSelection(t *testing.T) {
	got, err := parseSelectionArgs(`{"selection":2}`)
	if err != nil {
		t.Fatalf("parseSelectionArgs returned error: %v", err)
	}
	if got != "2" {
		t.Fatalf("parseSelectionArgs returned %q", got)
	}
}

func TestParseSelectionArgsNormalizesNoneOfTheseVariants(t *testing.T) {
	got, err := parseSelectionArgs(`{"selection":"none of these"}`)
	if err != nil {
		t.Fatalf("parseSelectionArgs returned error: %v", err)
	}
	if got != noneSelection {
		t.Fatalf("parseSelectionArgs returned %q", got)
	}
}

func TestNormalizeSelectionDoesNotAssumeOptionCount(t *testing.T) {
	got := normalizeSelection("1", 0)
	if got != "1" {
		t.Fatalf("normalizeSelection returned %q", got)
	}
}

func TestNormalizeSelectionMapsNumberedNoneToNoneOfThese(t *testing.T) {
	got := normalizeSelection("13", 12)
	if got != noneSelection {
		t.Fatalf("normalizeSelection returned %q", got)
	}
}

func TestDescribeCreateChatCompletionErrorForRequestError(t *testing.T) {
	err := describeCreateChatCompletionError(&openai.RequestError{
		HTTPStatusCode: 503,
		Err:            errors.New("invalid character 'u' looking for beginning of value"),
	})
	got := err.Error()
	want := "chat completion request failed before tool parsing: endpoint returned HTTP 503 with a non-JSON error response: invalid character 'u' looking for beginning of value"
	if got != want {
		t.Fatalf("describeCreateChatCompletionError returned %q", got)
	}
}

func TestDescribeCreateChatCompletionErrorForAPIError(t *testing.T) {
	err := describeCreateChatCompletionError(&openai.APIError{
		HTTPStatusCode: 503,
		Message:        "That model is currently overloaded",
	})
	got := err.Error()
	want := "chat completion request failed: endpoint returned HTTP 503: That model is currently overloaded"
	if got != want {
		t.Fatalf("describeCreateChatCompletionError returned %q", got)
	}
}
