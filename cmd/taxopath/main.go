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
	"taxowalk/internal/taxopath"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "taxopath:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		showVersion bool
		showMaximum bool
	)
	taxFlags := cmdutil.NewTaxonomyFlags()
	taxFlags.Register(flag.CommandLine)
	flag.BoolVar(&showVersion, "version", false, "print the taxopath version and exit")
	flag.BoolVar(&showMaximum, "maximum", false, "print the largest number used in any taxonomy path")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "taxopath - convert taxonomy IDs to dot-separated paths\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <taxonomy id>\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if showVersion {
		fmt.Printf("taxopath %s\n", cmdutil.ResolveVersion(version))
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tax, err := taxFlags.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to load taxonomy: %w", err)
	}

	if showMaximum {
		if flag.NArg() != 0 {
			return errors.New("-maximum does not take additional arguments")
		}
		max, err := taxopath.Maximum(tax)
		if err != nil {
			return err
		}
		fmt.Println(max)
		return nil
	}

	if flag.NArg() != 1 {
		return errors.New("a single taxonomy category ID must be provided")
	}
	id := strings.TrimSpace(flag.Arg(0))
	if id == "" {
		return errors.New("taxonomy category ID is empty")
	}

	node := tax.FindByID(id)
	if node == nil {
		return fmt.Errorf("taxonomy category %q not found", id)
	}

	path, err := taxopath.Path(id)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}
