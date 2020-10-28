package query

// Node is the programmatic representation of node(s) in GraphQL
type Node struct {
	name    string
	scalars []Scalar
	objects []Object
	unions  []Union
}

const (
	// TypeNode is the name of a "node"
	TypeNode = "Node"

	// TypeNodeList is the name of a "node list"
	TypeNodeList = "Nodes"
)

// NewNode creates and returns a pointer to a Node.
//
// The differences between a "node" and a "nodeList" is their naming and type in the final query struct.
// Node is named as "node" and takes a struct.
// Node list is named as "nodes" and takes a slice of structs.
func NewNode() *Node {
	return &Node{
		name: TypeNode,
	}
}

// NewNodeList creates and returns a pointer to a Node.
// The differences between a "node" and a "nodeList" is their naming and type in the final query struct.
// Node is named as "node" and takes a struct.
// Node list is named as "nodes" and takes a slice of structs.
func NewNodeList() *Node {
	return &Node{
		name: TypeNodeList,
	}
}

// AddScalar appends the given Scalar variable to its children
func (n *Node) AddScalar(scalar Scalar) {
	n.scalars = append(n.scalars, scalar)
}

// AddObject appends the given Object variable to its children
func (n *Node) AddObject(obj *Object) {
	n.objects = append(n.objects, *obj)
}

// AddUnion appends the given Union variable to its children
func (n *Node) AddUnion(u *Union) {
	n.unions = append(n.unions, *u)
}
