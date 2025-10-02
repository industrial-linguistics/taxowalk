package main

import (
	"flag"
	"fmt"
	"os"

	"taxowalk/internal/history"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "taxowalk-report:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		dbPath      string
		showAll     bool
		check24h    bool
		limitTokens int64
	)

	flag.StringVar(&dbPath, "db", "", "SQLite database path (required)")
	flag.BoolVar(&showAll, "all", false, "show all classification records")
	flag.BoolVar(&check24h, "check-24h", false, "check if token usage in last 24 hours exceeds limit")
	flag.Int64Var(&limitTokens, "limit", 5000000, "token limit for 24-hour check (default: 5000000)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "taxowalk-report - report on token usage from taxowalk history\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if dbPath == "" {
		return fmt.Errorf("database path (-db) is required")
	}

	db, err := history.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if check24h {
		return check24HourLimit(db, limitTokens)
	}

	if showAll {
		return showAllRecords(db)
	}

	return showSummary(db)
}

func showSummary(db *history.DB) error {
	total, err := db.GetTotalTokens()
	if err != nil {
		return err
	}

	last24h, err := db.GetTokensLast24Hours()
	if err != nil {
		return err
	}

	fmt.Printf("Token Usage Summary\n")
	fmt.Printf("===================\n")
	fmt.Printf("Total tokens (all time): %d\n", total)
	fmt.Printf("Tokens (last 24 hours):  %d\n", last24h)

	return nil
}

func showAllRecords(db *history.DB) error {
	records, err := db.GetAllRecords()
	if err != nil {
		return err
	}

	fmt.Printf("%-20s %-40s %-50s %-15s %10s %10s %10s\n",
		"Timestamp", "Product", "Category", "Category ID",
		"Prompt", "Compl", "Total")
	fmt.Println("------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------")

	for _, r := range records {
		productDesc := r.ProductDesc
		if len(productDesc) > 40 {
			productDesc = productDesc[:37] + "..."
		}
		category := r.Category
		if len(category) > 50 {
			category = category[:47] + "..."
		}
		categoryID := r.CategoryID
		if len(categoryID) > 15 {
			categoryID = categoryID[:12] + "..."
		}

		fmt.Printf("%-20s %-40s %-50s %-15s %10d %10d %10d\n",
			r.Timestamp.Format("2006-01-02 15:04:05"),
			productDesc,
			category,
			categoryID,
			r.PromptTokens,
			r.CompletionTokens,
			r.TotalTokens,
		)
	}

	total, _ := db.GetTotalTokens()
	fmt.Println("------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------")
	fmt.Printf("Total tokens: %d\n", total)

	return nil
}

func check24HourLimit(db *history.DB, limit int64) error {
	tokens, err := db.GetTokensLast24Hours()
	if err != nil {
		return err
	}

	fmt.Printf("Token usage in last 24 hours: %d\n", tokens)
	fmt.Printf("Limit: %d\n", limit)

	if tokens > limit {
		fmt.Printf("WARNING: Limit exceeded by %d tokens\n", tokens-limit)
		os.Exit(2)
	} else {
		fmt.Printf("OK: Within limit (%d tokens remaining)\n", limit-tokens)
	}

	return nil
}
