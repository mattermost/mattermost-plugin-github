package query

type Node struct {
	name    string
	scalars []Scalar
	objects []Object
}

func NewNode() *Node {
	return &Node{
		name: "nodes",
	}
}
