package taxopath

import (
	"fmt"
	"strconv"
	"strings"

	"taxowalk/internal/taxonomy"
)

const idPrefix = "gid://shopify/TaxonomyCategory/"

var topLevelNumbers = map[string]int{
	"aa": 1,  // Apparel & Accessories
	"ap": 2,  // Animals & Pet Supplies
	"ae": 3,  // Arts & Entertainment
	"bt": 4,  // Baby & Toddler
	"bu": 5,  // Bundles
	"bi": 6,  // Business & Industrial
	"co": 7,  // Cameras & Optics
	"el": 8,  // Electronics
	"fb": 9,  // Food, Beverages & Tobacco
	"fr": 10, // Furniture
	"gc": 11, // Gift Cards
	"ha": 12, // Hardware
	"hb": 13, // Health & Beauty
	"hg": 14, // Home & Garden
	"lb": 15, // Luggage & Bags
	"ma": 16, // Mature
	"me": 17, // Media
	"os": 18, // Office Supplies
	"pa": 19, // Product Add-Ons
	"rc": 20, // Religious & Ceremonial
	"se": 21, // Services
	"so": 22, // Software
	"sg": 23, // Sporting Goods
	"tg": 24, // Toys & Games
	"na": 25, // Uncategorized
	"vp": 26, // Vehicles & Parts
}

func Path(id string) (string, error) {
	prefix, segments, err := parseID(id)
	if err != nil {
		return "", err
	}
	root, ok := topLevelNumbers[prefix]
	if !ok {
		return "", fmt.Errorf("unknown taxonomy prefix %q", prefix)
	}
	parts := make([]string, 0, len(segments)+1)
	parts = append(parts, strconv.Itoa(root))
	for _, segment := range segments {
		n, err := strconv.Atoi(segment)
		if err != nil {
			return "", fmt.Errorf("invalid taxonomy segment %q", segment)
		}
		parts = append(parts, strconv.Itoa(n))
	}
	return strings.Join(parts, "."), nil
}

func Maximum(tax *taxonomy.Taxonomy) (int, error) {
	if tax == nil {
		return 0, fmt.Errorf("taxonomy is nil")
	}
	max := 0
	var walk func(node *taxonomy.Node) error
	walk = func(node *taxonomy.Node) error {
		if node == nil {
			return nil
		}
		if node.ID != "" {
			prefix, segments, err := parseID(node.ID)
			if err != nil {
				return err
			}
			root, ok := topLevelNumbers[prefix]
			if !ok {
				return fmt.Errorf("unknown taxonomy prefix %q", prefix)
			}
			if root > max {
				max = root
			}
			for _, segment := range segments {
				n, err := strconv.Atoi(segment)
				if err != nil {
					return fmt.Errorf("invalid taxonomy segment %q", segment)
				}
				if n > max {
					max = n
				}
			}
		}
		for _, child := range node.Children {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range tax.Roots {
		if err := walk(root); err != nil {
			return 0, err
		}
	}
	return max, nil
}

func parseID(id string) (string, []string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", nil, fmt.Errorf("taxonomy ID is empty")
	}
	if !strings.HasPrefix(trimmed, idPrefix) {
		return "", nil, fmt.Errorf("taxonomy ID %q does not use expected prefix", trimmed)
	}
	body := strings.TrimPrefix(trimmed, idPrefix)
	if body == "" {
		return "", nil, fmt.Errorf("taxonomy ID %q is missing identifier", trimmed)
	}
	parts := strings.Split(body, "-")
	prefix := parts[0]
	var segments []string
	if len(parts) > 1 {
		segments = parts[1:]
	}
	return prefix, segments, nil
}
