package taxonomy

import "testing"

func TestFindByID(t *testing.T) {
	tax := &Taxonomy{
		Roots: []*Node{
			{
				Name:     "Vertical",
				FullName: "Vertical",
				Children: []*Node{
					{
						ID:       "gid://shopify/TaxonomyCategory/aa",
						Name:     "Apparel & Accessories",
						FullName: "Apparel & Accessories",
						Children: []*Node{
							{
								ID:       "gid://shopify/TaxonomyCategory/aa-1",
								Name:     "Clothing",
								FullName: "Apparel & Accessories > Clothing",
							},
						},
					},
				},
			},
		},
	}

	node := tax.FindByID("gid://shopify/TaxonomyCategory/aa-1")
	if node == nil {
		t.Fatal("expected node to be found")
	}
	if node.Name != "Clothing" {
		t.Fatalf("unexpected node: %s", node.Name)
	}

	if tax.FindByID("gid://shopify/TaxonomyCategory/does-not-exist") != nil {
		t.Fatal("expected nil for unknown ID")
	}
}
