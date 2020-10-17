package query

import "fmt"

type Union Object

const TagKeyUnion = "union"

func NewUnion(name string) (*Union, error) {
	o, err := NewObject(SetName(name))
	if err != nil {
		return nil, err
	}

	o.tag[TagKeyUnion] = "... on " + o.name
	u := Union(*o)

	return &u, nil
}

func (u *Union) AddScalar(scalar Scalar) {
	u.scalars = append(u.scalars, scalar)
}

func (u *Union) AddObject(obj *Object) {
	u.objects = append(u.objects, *obj)
}

func (u *Union) SetNode(n *Node) error {
	if n.name == TypeNodeList {
		return fmt.Errorf("cannot set node list to node")
	}

	u.node = n
	return nil
}

func (u *Union) SetNodeList(n *Node) error {
	if n.name == TypeNode {
		return fmt.Errorf("cannot set node to node list")
	}

	u.nodeList = n
	return nil
}
