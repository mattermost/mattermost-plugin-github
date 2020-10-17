package query

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

func (u *Union) SetNode(n *Node) {
	u.node = n
}
