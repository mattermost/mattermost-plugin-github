package query

type Node struct {
	name    string
	scalars []Scalar
	objects []Object
	unions  []Union
}

const (
	TypeNode     = "Node"
	TypeNodeList = "Nodes"
)

func NewNode() *Node {
	return &Node{
		name: TypeNode,
	}
}

func NewNodeList() *Node {
	return &Node{
		name: TypeNodeList,
	}
}

func (n *Node) AddScalar(scalar Scalar) {
	n.scalars = append(n.scalars, scalar)
}

func (n *Node) AddObject(obj *Object) {
	n.objects = append(n.objects, *obj)
}

func (n *Node) AddUnion(u *Union) {
	n.unions = append(n.unions, *u)
}
