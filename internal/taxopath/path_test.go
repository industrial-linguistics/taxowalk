package taxopath

import (
	"testing"

	"taxowalk/internal/taxonomy"
)

func TestPath(t *testing.T) {
	cases := map[string]string{
		"gid://shopify/TaxonomyCategory/aa":        "1",
		"gid://shopify/TaxonomyCategory/aa-1":      "1.1",
		"gid://shopify/TaxonomyCategory/aa-1-13-8": "1.1.13.8",
		"gid://shopify/TaxonomyCategory/ap-2-1":    "2.2.1",
	}
	for id, expected := range cases {
		path, err := Path(id)
		if err != nil {
			t.Fatalf("Path(%q) returned error: %v", id, err)
		}
		if path != expected {
			t.Fatalf("Path(%q) = %q, expected %q", id, path, expected)
		}
	}

	if _, err := Path("gid://shopify/TaxonomyCategory/zz-1"); err == nil {
		t.Fatal("expected error for unknown prefix")
	}
}

func TestMaximum(t *testing.T) {
	tax := &taxonomy.Taxonomy{
		Roots: []*taxonomy.Node{
			{
				Children: []*taxonomy.Node{
					{
						ID: "gid://shopify/TaxonomyCategory/aa",
						Children: []*taxonomy.Node{
							{ID: "gid://shopify/TaxonomyCategory/aa-1"},
							{ID: "gid://shopify/TaxonomyCategory/aa-1-27"},
						},
					},
					{
						ID: "gid://shopify/TaxonomyCategory/vp-99",
					},
				},
			},
		},
	}
	max, err := Maximum(tax)
	if err != nil {
		t.Fatalf("Maximum returned error: %v", err)
	}
	if max != 99 {
		t.Fatalf("Maximum = %d, expected 99", max)
	}
}
