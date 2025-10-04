package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"taxowalk/internal/cmdutil"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "taxoname:", err)
		os.Exit(1)
	}
}

func run() error {
	var showVersion bool
	taxFlags := cmdutil.NewTaxonomyFlags()
	taxFlags.Register(flag.CommandLine)
	flag.BoolVar(&showVersion, "version", false, "print the taxoname version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "taxoname - resolve taxonomy IDs to their full path\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <taxonomy id>\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if showVersion {
		fmt.Printf("taxoname %s\n", cmdutil.ResolveVersion(version))
		return nil
	}

	if flag.NArg() != 1 {
		return errors.New("a single taxonomy category ID must be provided")
	}

	id := strings.TrimSpace(flag.Arg(0))
	if id == "" {
		return errors.New("taxonomy category ID is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tax, err := taxFlags.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to load taxonomy: %w", err)
	}

	node := tax.FindByID(id)
	if node == nil {
		return fmt.Errorf("taxonomy category %q not found", id)
	}
	if node.FullName != "" {
		fmt.Println(node.FullName)
	} else {
		fmt.Println(node.Name)
	}
	return nil
}
