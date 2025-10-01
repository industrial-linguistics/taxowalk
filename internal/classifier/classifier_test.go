package classifier

import (
	"context"
	"errors"
	"testing"

	"bulksense/internal/llm"
	"bulksense/internal/taxonomy"
)

type mockModel struct {
	responses []string
	call      int
	err       error
}

func (m *mockModel) ChooseOption(ctx context.Context, prompt llm.Prompt) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.call >= len(m.responses) {
		return "none of these", nil
	}
	resp := m.responses[m.call]
	m.call++
	return resp, nil
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
