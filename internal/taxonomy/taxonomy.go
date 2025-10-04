package taxonomy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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

const cacheMaxAge = 24 * time.Hour

var errCacheMiss = errors.New("cache miss")

type fetchConfig struct {
	disableCache bool
}

// FetchOption configures Fetch behaviour.
type FetchOption func(*fetchConfig)

// WithCacheDisabled disables the taxonomy cache when fetching.
func WithCacheDisabled() FetchOption {
	return func(cfg *fetchConfig) {
		cfg.disableCache = true
	}
}

func Fetch(ctx context.Context, source string, opts ...FetchOption) (*Taxonomy, error) {
	if source == "" {
		return nil, errors.New("taxonomy source is empty")
	}
	cfg := fetchConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	u, err := url.Parse(source)
	if err == nil && u.Scheme != "" && u.Scheme != "file" {
		if !cfg.disableCache {
			if tax, err := loadFromCache(source); err == nil {
				return tax, nil
			}
		}
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
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		tax, err := decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		if !cfg.disableCache {
			saveToCache(source, data)
		}
		return tax, nil
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
		vertNode := &Node{Name: vertical.Name, FullName: vertical.Name, Children: []*Node{}}
		for _, cat := range vertical.Categories {
			vertNode.Children = append(vertNode.Children, convert(cat))
		}
		tax.Roots = append(tax.Roots, vertNode)
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

func cacheFilePath(source string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(source))
	name := hex.EncodeToString(sum[:]) + ".json"
	return filepath.Join(dir, "taxowalk", name), nil
}

func loadFromCache(source string) (*Taxonomy, error) {
	path, err := cacheFilePath(source)
	if err != nil {
		return nil, errCacheMiss
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, errCacheMiss
	}
	if time.Since(info.ModTime()) > cacheMaxAge {
		return nil, errCacheMiss
	}
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, errCacheMiss
	}
	defer f.Close()
	tax, err := decode(f)
	if err != nil {
		return nil, errCacheMiss
	}
	return tax, nil
}

func saveToCache(source string, data []byte) {
	path, err := cacheFilePath(source)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "taxonomy-*.json")
	if err != nil {
		return
	}
	defer tmp.Close()
	if _, err := tmp.Write(data); err != nil {
		os.Remove(tmp.Name())
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return
	}
	_ = os.Rename(tmp.Name(), path)
}
