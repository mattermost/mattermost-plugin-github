package query

import (
	"fmt"
	"strings"
)

// Scalar is the programmatic representation of scalar types in GraphQL.
// Name and type of Scalar must be the same with GitHub GraphQL API except for the "!" at the end.
// For more information about GraphQL scalar type see: https://graphql.org/learn/schema/#scalar-types
type Scalar struct {
	name string
	kind string
}

// NewScalar creates and returns a Scalar
// Values below are checked and converted to uppercase:
//	- ID
//	- URL
//	- URI
func NewScalar(name, kind string) (Scalar, error) {
	if err := validKey(name); err != nil {
		return Scalar{}, fmt.Errorf("error setting 'key': %s", err.Error())
	}
	if err := validKey(kind); err != nil {
		return Scalar{}, fmt.Errorf("error setting 'kind': %s", err.Error())
	}

	name = strings.TrimSpace(strings.Title(name))
	if strings.EqualFold(name, "id") || strings.EqualFold(name, "url") {
		name = strings.ToUpper(name)
	}

	kind = strings.TrimSpace(strings.Title(kind))
	if strings.EqualFold(kind, "id") || strings.EqualFold(kind, "uri") {
		kind = strings.ToUpper(kind)
	}

	return Scalar{name: name, kind: kind}, nil
}

// NewScalarGroup generates multiple Scalar item at once and returns them in a slice.
// Keys of parameter sg: name of the Scalar item
// Values of parameter sg: type of the Scalar item
func NewScalarGroup(sg map[string]string) ([]Scalar, error) {
	var res []Scalar
	for k, v := range sg {
		s, err := NewScalar(k, v)
		if err != nil {
			return nil, fmt.Errorf("error creating [%s: %s]: %v", k, v, err)
		}
		res = append(res, s)
	}
	return res, nil
}
