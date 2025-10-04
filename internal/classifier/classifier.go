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
	model      llm.Model
	taxonomy   *taxonomy.Taxonomy
	totalUsage llm.Usage
	debugf     func(format string, args ...interface{})
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

func (c *Classifier) SetDebugLogger(fn func(format string, args ...interface{})) {
	c.debugf = fn
}

func (c *Classifier) logf(format string, args ...interface{}) {
	if c != nil && c.debugf != nil {
		c.debugf(format, args...)
	}
}

func (c *Classifier) Classify(ctx context.Context, description string) (*taxonomy.Node, error) {
	if strings.TrimSpace(description) == "" {
		return nil, errors.New("description is empty")
	}

	c.totalUsage = llm.Usage{}
	var current *taxonomy.Node
	options := c.taxonomy.Roots
	var path []string

	c.logf("Starting classification with %d root options", len(options))

	for {
		available := nextLevelOptions(current, options)
		if len(available) == 0 {
			break
		}
		if len(path) > 0 {
			c.logf("Current path: %s", strings.Join(path, " > "))
		} else {
			c.logf("Current path: <root>")
		}
		prompt := llm.Prompt{
			Description: description,
			Path:        append([]string{}, path...),
			Options:     make([]llm.Option, len(available)),
		}
		for i, opt := range available {
			prompt.Options[i] = llm.Option{Name: opt.Name, FullName: opt.FullName, ID: opt.ID}
		}

		optionSummaries := make([]string, len(available))
		for i, opt := range available {
			optionSummaries[i] = fmt.Sprintf("%s (%s)", opt.FullName, opt.ID)
		}
		c.logf("Candidate options: %s", strings.Join(optionSummaries, "; "))

		c.logf("Requesting model choice")
		result, err := c.model.ChooseOption(ctx, prompt)
		if err != nil {
			return nil, err
		}

		c.logf("Model returned choice %q (prompt tokens: %d, completion tokens: %d, total: %d)",
			result.Choice, result.Usage.PromptTokens, result.Usage.CompletionTokens, result.Usage.TotalTokens)

		c.totalUsage.PromptTokens += result.Usage.PromptTokens
		c.totalUsage.CompletionTokens += result.Usage.CompletionTokens
		c.totalUsage.TotalTokens += result.Usage.TotalTokens

		normalized := normalizeChoice(result.Choice)
		if strings.EqualFold(normalized, "none of these") {
			c.logf("Model selected 'none of these'; stopping classification")
			break
		}
		var next *taxonomy.Node
		for _, opt := range available {
			if strings.EqualFold(opt.Name, normalized) || strings.EqualFold(opt.FullName, normalized) || strings.EqualFold(opt.ID, normalized) {
				next = opt
				break
			}
		}
		if next == nil {
			c.logf("Model choice %q did not match any candidate option", result.Choice)
			return current, fmt.Errorf("model selected unknown option %q", result.Choice)
		}

		current = next
		path = append(path, current.Name)
		options = current.Children
		c.logf("Descending to %s (%s) with %d child options", current.FullName, current.ID, len(nextLevelOptions(current, options)))
	}

	if current == nil {
		c.logf("No matching category identified")
	} else {
		c.logf("Final classification: %s (%s)", current.FullName, current.ID)
	}
	return current, nil
}

func (c *Classifier) Usage() llm.Usage {
	return c.totalUsage
}

func normalizeChoice(choice string) string {
	normalized := strings.TrimSpace(choice)
	if normalized == "" {
		return normalized
	}
	lower := strings.ToLower(normalized)
	if idx := strings.LastIndex(lower, " (id:"); idx != -1 {
		normalized = strings.TrimSpace(normalized[:idx])
	}
	return normalized
}

func nextLevelOptions(parent *taxonomy.Node, options []*taxonomy.Node) []*taxonomy.Node {
	if len(options) == 0 {
		return options
	}
	if parent == nil {
		return options
	}
	parentDepth := optionDepth(parent.ID)
	minDepth := 0
	for _, opt := range options {
		depth := optionDepth(opt.ID)
		if depth <= parentDepth {
			continue
		}
		if minDepth == 0 || depth < minDepth {
			minDepth = depth
		}
	}
	if minDepth == 0 {
		return options
	}
	filtered := make([]*taxonomy.Node, 0, len(options))
	for _, opt := range options {
		if optionDepth(opt.ID) == minDepth {
			filtered = append(filtered, opt)
		}
	}
	return filtered
}

func optionDepth(id string) int {
	if id == "" {
		return 0
	}
	return strings.Count(id, "-") + 1
}
