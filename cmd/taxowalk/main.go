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
	"taxowalk/internal/history"
	"taxowalk/internal/llm"
	"taxowalk/internal/taxonomy"
)

const defaultTaxonomyURL = "https://raw.githubusercontent.com/Shopify/product-taxonomy/refs/heads/main/dist/en/taxonomy.json"

var debugEnabled bool

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
		dbPath      string
	)

	flag.BoolVar(&useStdin, "stdin", false, "read the product description from standard input")
	flag.StringVar(&apiKeyFlag, "openai-key", "", "OpenAI API key (overrides defaults)")
	flag.StringVar(&taxonomyURL, "taxonomy-url", defaultTaxonomyURL, "URL or file path for the Shopify taxonomy JSON")
	flag.StringVar(&baseURL, "openai-base-url", "", "override the OpenAI API base URL")
	flag.StringVar(&dbPath, "history-db", "", "SQLite database path to track token usage history")
	flag.BoolVar(&debugEnabled, "debug", false, "enable verbose debug logging to standard error")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "taxowalk - classify products into the Shopify taxonomy\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] [product description]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	debugf("Arguments: %s", strings.Join(os.Args[1:], " "))

	description, err := loadDescription(useStdin, flag.Args())
	if err != nil {
		return err
	}

	debugf("Product description (%d chars)", len(description))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	debugf("Fetching taxonomy from %s", taxonomyURL)
	tax, err := taxonomy.Fetch(ctx, taxonomyURL)
	if err != nil {
		return fmt.Errorf("failed to load taxonomy: %w", err)
	}
	debugf("Fetched taxonomy in %s (%d root categories)", time.Since(start), len(tax.Roots))

	apiKey, err := resolveAPIKey(apiKeyFlag)
	if err != nil {
		return err
	}
	debugf("Resolved API key")

	var opts []llm.OptionFunc
	if baseURL != "" {
		opts = append(opts, llm.WithBaseURL(baseURL))
		debugf("Using custom OpenAI base URL: %s", baseURL)
	}
	model, err := llm.NewOpenAIModel(apiKey, opts...)
	if err != nil {
		return err
	}
	debugf("Initialised OpenAI model")

	clf, err := classifier.New(model, tax)
	if err != nil {
		return err
	}

	if debugEnabled {
		clf.SetDebugLogger(func(format string, args ...interface{}) {
			debugf("classifier: "+format, args...)
		})
	}

	node, err := clf.Classify(ctx, description)
	if err != nil {
		return err
	}

	usage := clf.Usage()
	debugf("Token usage - prompt: %d, completion: %d, total: %d", usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)

	if dbPath != "" {
		debugf("Recording classification history in %s", dbPath)
		db, err := history.Open(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to open history database: %v\n", err)
		} else {
			defer db.Close()
			categoryName := ""
			categoryID := ""
			if node != nil {
				categoryName = node.FullName
				categoryID = node.ID
			}
			if err := db.RecordClassification(description, categoryName, categoryID,
				usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record classification: %v\n", err)
			} else {
				debugf("Classification history recorded")
			}
		}
	}

	if node == nil {
		debugf("Classifier returned nil node")
		fmt.Println("No matching Shopify category found.")
		return nil
	}

	debugf("Classification result: %s (%s)", node.FullName, node.ID)
	fmt.Printf("%s\n%s\n", node.FullName, node.ID)
	return nil
}

func loadDescription(useStdin bool, args []string) (string, error) {
	if useStdin {
		debugf("Reading product description from standard input")
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
	debugf("Using product description from command line arguments (%d parts)", len(args))
	return strings.TrimSpace(strings.Join(args, " ")), nil
}

func resolveAPIKey(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		debugf("Using API key provided via --openai-key flag")
		return strings.TrimSpace(explicit), nil
	}
	if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" {
		debugf("Using API key from OPENAI_API_KEY environment variable")
		return key, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home directory: %w", err)
	}
	path := filepath.Join(home, ".openai.key")
	debugf("Reading API key from %s", path)
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

func debugf(format string, args ...interface{}) {
	if !debugEnabled {
		return
	}
	fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
}
