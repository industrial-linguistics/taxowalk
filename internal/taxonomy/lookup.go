package taxonomy

import "strings"

func (t *Taxonomy) FindByID(id string) *Node {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	for _, root := range t.Roots {
		if node := findByID(root, id); node != nil {
			return node
		}
	}
	return nil
}

func findByID(node *Node, id string) *Node {
	if node == nil {
		return nil
	}
	if node.ID == id {
		return node
	}
	for _, child := range node.Children {
		if found := findByID(child, id); found != nil {
			return found
		}
	}
	return nil
}
