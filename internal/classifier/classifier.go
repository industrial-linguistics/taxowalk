package classifier

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"taxowalk/internal/llm"
	"taxowalk/internal/taxonomy"
)

type Classifier struct {
	model    llm.Model
	taxonomy *taxonomy.Taxonomy
}

func New(model llm.Model, tax *taxonomy.Taxonomy) (*Classifier, error) {
	if model == nil {
		return nil, errors.New("model cannot be nil")
	}
	if tax == nil || len(tax.Roots) == 0 {
		return nil, errors.New("taxonomy cannot be nil or empty")
	}
	return &Classifier{model: model, taxonomy: tax}, nil
}

func (c *Classifier) Classify(ctx context.Context, description string) (*taxonomy.Node, error) {
	if strings.TrimSpace(description) == "" {
		return nil, errors.New("description is empty")
	}

	var current *taxonomy.Node
	options := c.taxonomy.Roots
	var path []string

	for len(options) > 0 {
		prompt := llm.Prompt{
			Description: description,
			Path:        append([]string{}, path...),
			Options:     make([]llm.Option, len(options)),
		}
		for i, opt := range options {
			prompt.Options[i] = llm.Option{Name: opt.Name, FullName: opt.FullName, ID: opt.ID}
		}

		choice, err := c.model.ChooseOption(ctx, prompt)
		if err != nil {
			return nil, err
		}
		normalized := strings.TrimSpace(choice)
		if strings.EqualFold(normalized, "none of these") {
			break
		}
		var next *taxonomy.Node
		for _, opt := range options {
			if strings.EqualFold(opt.Name, normalized) || strings.EqualFold(opt.FullName, normalized) || strings.EqualFold(opt.ID, normalized) {
				next = opt
				break
			}
		}
		if next == nil {
			return current, fmt.Errorf("model selected unknown option %q", choice)
		}

		current = next
		path = append(path, current.Name)
		options = current.Children
	}

	return current, nil
}
