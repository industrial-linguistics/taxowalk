package cmdutil

import (
	"context"
	"flag"

	"taxowalk/internal/taxonomy"
)

const DefaultTaxonomyURL = "https://raw.githubusercontent.com/Shopify/product-taxonomy/refs/heads/main/dist/en/taxonomy.json"

type TaxonomyFlags struct {
	URL     string
	Refresh bool
}

func NewTaxonomyFlags() TaxonomyFlags {
	return TaxonomyFlags{URL: DefaultTaxonomyURL}
}

func (f *TaxonomyFlags) Register(fs *flag.FlagSet) {
	if f.URL == "" {
		f.URL = DefaultTaxonomyURL
	}
	fs.StringVar(&f.URL, "taxonomy-url", f.URL, "URL or file path for the Shopify taxonomy JSON")
	fs.BoolVar(&f.Refresh, "refresh-taxonomy", false, "ignore cached taxonomy data and fetch a fresh copy")
}

func (f *TaxonomyFlags) Fetch(ctx context.Context) (*taxonomy.Taxonomy, error) {
	var opts []taxonomy.FetchOption
	if f.Refresh {
		opts = append(opts, taxonomy.WithCacheDisabled())
	}
	return taxonomy.Fetch(ctx, f.URL, opts...)
}
