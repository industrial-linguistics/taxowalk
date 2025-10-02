package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"taxowalk/internal/classifier"
	"taxowalk/internal/llm"
	"taxowalk/internal/taxonomy"
)

const defaultTaxonomyURL = "https://raw.githubusercontent.com/Shopify/product-taxonomy/refs/heads/main/dist/en/taxonomy.json"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "taxowalk:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		useStdin    bool
		apiKeyFlag  string
		taxonomyURL string
		baseURL     string
	)

	flag.BoolVar(&useStdin, "stdin", false, "read the product description from standard input")
	flag.StringVar(&apiKeyFlag, "openai-key", "", "OpenAI API key (overrides defaults)")
	flag.StringVar(&taxonomyURL, "taxonomy-url", defaultTaxonomyURL, "URL or file path for the Shopify taxonomy JSON")
	flag.StringVar(&baseURL, "openai-base-url", "", "override the OpenAI API base URL")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "taxowalk - classify products into the Shopify taxonomy\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] [product description]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	description, err := loadDescription(useStdin, flag.Args())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tax, err := taxonomy.Fetch(ctx, taxonomyURL)
	if err != nil {
		return fmt.Errorf("failed to load taxonomy: %w", err)
	}

	apiKey, err := resolveAPIKey(apiKeyFlag)
	if err != nil {
		return err
	}

	var opts []llm.OptionFunc
	if baseURL != "" {
		opts = append(opts, llm.WithBaseURL(baseURL))
	}
	model, err := llm.NewOpenAIModel(apiKey, opts...)
	if err != nil {
		return err
	}

	clf, err := classifier.New(model, tax)
	if err != nil {
		return err
	}

	node, err := clf.Classify(ctx, description)
	if err != nil {
		return err
	}
	if node == nil {
		fmt.Println("No matching Shopify category found.")
		return nil
	}

	fmt.Printf("%s\n%s\n", node.FullName, node.ID)
	return nil
}

func loadDescription(useStdin bool, args []string) (string, error) {
	if useStdin {
		info, err := os.Stdin.Stat()
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return "", errors.New("--stdin specified but no data piped in")
		}
		data, err := io.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	if len(args) == 0 {
		return "", errors.New("no product description provided")
	}
	return strings.TrimSpace(strings.Join(args, " ")), nil
}

func resolveAPIKey(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" {
		return key, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home directory: %w", err)
	}
	path := filepath.Join(home, ".openai.key")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read API key from %s: %w", path, err)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("API key file %s is empty", path)
	}
	return key, nil
}
