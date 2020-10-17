package query

type Node struct {
	name    string
	scalars []Scalar
	objects []Object
}

func NewNode() *Node {
	return &Node{
		name: "Nodes",
	}
}

func (n *Node) AddScalar(scalar Scalar) {
	n.scalars = append(n.scalars, scalar)
}

func (n *Node) AddObject(obj *Object) {
	n.objects = append(n.objects, *obj)
}
