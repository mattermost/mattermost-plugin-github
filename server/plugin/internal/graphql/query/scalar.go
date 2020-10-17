package query

import (
	"fmt"
	"strings"
)

type Scalar struct {
	name string
	kind string
}

func NewScalar(name, kind string) (Scalar, error) {
	if err := validKey(name); err != nil {
		return Scalar{}, fmt.Errorf("error setting 'key': %s", err.Error())
	}
	if err := validKey(kind); err != nil {
		return Scalar{}, fmt.Errorf("error setting 'kind': %s", err.Error())
	}

	name = strings.TrimSpace(strings.Title(name))
	if strings.ToLower(name) == "id" {
		name = "ID"
	}

	kind = strings.TrimSpace(strings.Title(kind))
	if strings.ToLower(kind) == "id" {
		kind = "ID"
	}

	return Scalar{name: name, kind: kind}, nil
}
