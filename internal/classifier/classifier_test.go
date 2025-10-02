package classifier

import (
	"context"
	"errors"
	"testing"

	"taxowalk/internal/llm"
	"taxowalk/internal/taxonomy"
)

type mockModel struct {
	responses []string
	call      int
	err       error
}

func (m *mockModel) ChooseOption(ctx context.Context, prompt llm.Prompt) (*llm.Result, error) {
	if m.err != nil {
		return nil, m.err
	}
	choice := "none of these"
	if m.call < len(m.responses) {
		choice = m.responses[m.call]
		m.call++
	}
	return &llm.Result{
		Choice: choice,
		Usage:  llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}, nil
}

func TestClassifierWalksTree(t *testing.T) {
	root := &taxonomy.Node{ID: "root", Name: "Root", FullName: "Root"}
	child := &taxonomy.Node{ID: "child", Name: "Child", FullName: "Root > Child"}
	root.Children = []*taxonomy.Node{child}
	tax := &taxonomy.Taxonomy{Version: "test", Roots: []*taxonomy.Node{root}}

	model := &mockModel{responses: []string{"Root", "none of these"}}
	clf, err := New(model, tax)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	node, err := clf.Classify(context.Background(), "example")
	if err != nil {
		t.Fatalf("Classify returned error: %v", err)
	}
	if node != root {
		t.Fatalf("expected root, got %#v", node)
	}
}

func TestClassifierSelectsChild(t *testing.T) {
	root := &taxonomy.Node{ID: "root", Name: "Root", FullName: "Root"}
	child := &taxonomy.Node{ID: "child", Name: "Child", FullName: "Root > Child"}
	root.Children = []*taxonomy.Node{child}
	tax := &taxonomy.Taxonomy{Version: "test", Roots: []*taxonomy.Node{root}}

	model := &mockModel{responses: []string{"Root", "Child"}}
	clf, err := New(model, tax)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	node, err := clf.Classify(context.Background(), "example")
	if err != nil {
		t.Fatalf("Classify returned error: %v", err)
	}
	if node != child {
		t.Fatalf("expected child, got %#v", node)
	}
}

func TestClassifierPropagatesModelError(t *testing.T) {
	root := &taxonomy.Node{ID: "root", Name: "Root", FullName: "Root"}
	tax := &taxonomy.Taxonomy{Version: "test", Roots: []*taxonomy.Node{root}}
	model := &mockModel{err: errors.New("boom")}

	clf, err := New(model, tax)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := clf.Classify(context.Background(), "example"); err == nil {
		t.Fatalf("expected error from model")
	}
}
