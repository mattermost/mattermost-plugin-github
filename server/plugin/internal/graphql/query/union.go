package query

import "fmt"

// Union is the programmatic representation of union types in GraphQL.
// For more information about GraphQL union type see: https://graphql.org/learn/schema/#union-types
type Union Object

// NewUnion creates and returns pointer to a Union.
// Creation of Union is not any different than an Object's so this constructor creates an object and converts it to Union.
// Example:
// NewUnion("PullRequest") crates [... on PullRequest{...}]
func NewUnion(name string) (*Union, error) {
	o, err := NewObject(name)
	if err != nil {
		return nil, fmt.Errorf("error creating new Union type: %v", err)
	}

	u := Union(*o)

	return &u, nil
}

// AddScalar appends the given Scalar variable to its children
func (u *Union) AddScalar(scalar Scalar) {
	u.scalars = append(u.scalars, scalar)
}

// AddScalarGroup appends the given Scalar slice to its children
func (u *Union) AddScalarGroup(scalars []Scalar) {
	u.scalars = append(u.scalars, scalars...)
}

// AddObject appends the given Object variable to its children
func (u *Union) AddObject(obj *Object) {
	u.objects = append(u.objects, *obj)
}
