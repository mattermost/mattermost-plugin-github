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
