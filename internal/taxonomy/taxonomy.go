package taxonomy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Taxonomy struct {
	Version string
	Roots   []*Node
}

type Node struct {
	ID       string
	Name     string
	FullName string
	Children []*Node
}

func (n *Node) FindChildByName(name string) *Node {
	trimmed := strings.TrimSpace(name)
	for _, child := range n.Children {
		if strings.EqualFold(child.Name, trimmed) || strings.EqualFold(child.FullName, trimmed) {
			return child
		}
	}
	return nil
}

type Option struct {
	Name     string
	FullName string
	ID       string
}

func (n *Node) Options() []Option {
	opts := make([]Option, len(n.Children))
	for i, child := range n.Children {
		opts[i] = Option{Name: child.Name, FullName: child.FullName, ID: child.ID}
	}
	return opts
}

type rawTaxonomy struct {
	Version   string        `json:"version"`
	Verticals []rawVertical `json:"verticals"`
}

type rawVertical struct {
	Name       string        `json:"name"`
	Categories []rawCategory `json:"categories"`
}

type rawCategory struct {
	ID       string        `json:"id"`
	Level    int           `json:"level"`
	Name     string        `json:"name"`
	FullName string        `json:"full_name"`
	Children []rawCategory `json:"children"`
}

func Fetch(ctx context.Context, source string) (*Taxonomy, error) {
	if source == "" {
		return nil, errors.New("taxonomy source is empty")
	}
	u, err := url.Parse(source)
	if err == nil && u.Scheme != "" && u.Scheme != "file" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch taxonomy: %s", resp.Status)
		}
		return decode(resp.Body)
	}

	// Treat as file path.
	path := source
	if u != nil && u.Scheme == "file" {
		path = u.Path
	}
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return decode(f)
}

func decode(r io.Reader) (*Taxonomy, error) {
	var raw rawTaxonomy
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, err
	}
	tax := &Taxonomy{Version: raw.Version}
	for _, vertical := range raw.Verticals {
		for _, cat := range vertical.Categories {
			node := convert(cat)
			tax.Roots = append(tax.Roots, node)
		}
	}
	if len(tax.Roots) == 0 {
		return nil, errors.New("taxonomy has no root categories")
	}
	return tax, nil
}

func convert(cat rawCategory) *Node {
	node := &Node{ID: cat.ID, Name: cat.Name, FullName: cat.FullName}
	if len(cat.Children) == 0 {
		node.Children = []*Node{}
		return node
	}
	node.Children = make([]*Node, len(cat.Children))
	for i, child := range cat.Children {
		node.Children[i] = convert(child)
	}
	return node
}
