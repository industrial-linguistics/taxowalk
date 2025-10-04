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
	prompts   []llm.Prompt
}

func (m *mockModel) ChooseOption(ctx context.Context, prompt llm.Prompt) (*llm.Result, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.prompts = append(m.prompts, prompt)
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

func TestClassifierOffersOnlyNextLevelOptions(t *testing.T) {
	leaf := &taxonomy.Node{ID: "aa-1", Name: "Sub", FullName: "Top > Category > Sub"}
	grandChild := &taxonomy.Node{ID: "aa-1-1", Name: "SubSub", FullName: "Top > Category > Sub > SubSub"}
	parent := &taxonomy.Node{ID: "aa", Name: "Category", FullName: "Top > Category", Children: []*taxonomy.Node{leaf, grandChild}}
	root := &taxonomy.Node{ID: "", Name: "Top", FullName: "Top", Children: []*taxonomy.Node{parent}}
	tax := &taxonomy.Taxonomy{Version: "test", Roots: []*taxonomy.Node{root}}

	model := &mockModel{responses: []string{"Top", "Category", "Sub"}}
	clf, err := New(model, tax)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	node, err := clf.Classify(context.Background(), "example")
	if err != nil {
		t.Fatalf("Classify returned error: %v", err)
	}
	if node != leaf {
		t.Fatalf("expected leaf node, got %#v", node)
	}
	if len(model.prompts) < 3 {
		t.Fatalf("expected at least 3 prompts, got %d", len(model.prompts))
	}
	last := model.prompts[2]
	if len(last.Options) != 1 {
		t.Fatalf("expected 1 option at third level, got %d", len(last.Options))
	}
	if last.Options[0].FullName != leaf.FullName {
		t.Fatalf("unexpected option at third level: %#v", last.Options[0])
	}
}

func TestNormalizeChoiceStripsIDAnnotation(t *testing.T) {
	input := "Apparel & Accessories > Clothing > Tops (id: gid://example)"
	if got := normalizeChoice(input); got != "Apparel & Accessories > Clothing > Tops" {
		t.Fatalf("normalizeChoice returned %q", got)
	}
}
