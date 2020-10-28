package query

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/internal/graphql/query/decorator"
)

type Builder struct {
	root *Object
}

// NewBuilder creates and returns a Builder
func NewBuilder(root *Object) Builder {
	return Builder{root: root}
}

// Build generates and returns the query struct
func (b Builder) Build() (interface{}, error) {
	queryFields, err := buildObjectQuery([]Object{*b.root})
	if err != nil {
		return nil, fmt.Errorf("error building query: %v", err)
	}

	return reflect.New(reflect.StructOf(queryFields)).Elem().Interface(), nil
}

func buildScalarQuery(scalars []Scalar) ([]reflect.StructField, error) {
	var fields []reflect.StructField

	for _, s := range scalars {
		field := reflect.StructField{
			Name: s.name,
		}

		if err := decorator.DecorateScalarType(s.kind, &field.Type); err != nil {
			return nil, fmt.Errorf("error building scalar query: %v", err)
		}

		fields = append(fields, field)
	}

	return fields, nil
}

func buildUnionQuery(unions []Union) ([]reflect.StructField, error) {
	var fields []reflect.StructField

	for _, u := range unions {
		children, err := buildScalarQuery(u.scalars)
		if err != nil {
			return nil, fmt.Errorf("error building scalar query at child node: %+v\nerr: %v", u.scalars, err)
		}

		if len(u.objects) > 0 {
			oq, err := buildObjectQuery(u.objects)
			if err != nil {
				return nil, fmt.Errorf("error building object query at child node: %v", err)
			}
			children = append(children, oq...)
		}

		field := reflect.StructField{
			Name: u.name,
			Type: reflect.StructOf(children),
			Tag:  reflect.StructTag(fmt.Sprintf(`graphql:"... on %s"`, u.name)),
		}

		fields = append(fields, field)
	}

	return fields, nil
}

func buildObjectQuery(objs []Object) ([]reflect.StructField, error) {
	var fields []reflect.StructField

	for _, o := range objs {
		// Object query must have children. Return error no children was set
		if len(o.scalars)+len(o.objects)+len(o.unions) == 0 && o.nodeList == nil && o.node == nil {
			return nil, fmt.Errorf("object query must have at least one of the following: node, nodeList, scalar, objet")
		}

		var children []reflect.StructField

		if len(o.scalars) > 0 {
			scalars, err := buildScalarQuery(o.scalars)
			if err != nil {
				return nil, fmt.Errorf("error building scalar query at child node: %+v\nerr: %v", o.scalars, err)
			}

			children = append(children, scalars...)
		}

		if len(o.objects) > 0 {
			oq, err := buildObjectQuery(o.objects)
			if err != nil {
				return nil, fmt.Errorf("failed at building nested object query: %v", err)
			}

			children = append(children, oq...)
		}

		if o.node != nil {
			nq, err := buildNode(o.node)
			if err != nil {
				return nil, fmt.Errorf("failed at building nested node query: %v", err)
			}

			children = append(children, nq)
		}

		if o.nodeList != nil {
			nq, err := buildNode(o.nodeList)
			if err != nil {
				return nil, fmt.Errorf("failed at building nested node list query: %v", err)
			}

			children = append(children, nq)
		}

		if len(o.unions) > 0 {
			uq, err := buildUnionQuery(o.unions)
			if err != nil {
				return nil, fmt.Errorf("failed at building nested union query: %v", err)
			}
			children = append(children, uq...)
		}

		query := reflect.StructField{
			Name: o.name,
			Type: reflect.StructOf(children),
		}

		if fieldTag := buildTag(o.tag, o.name); fieldTag != "" {
			query.Tag = reflect.StructTag(fieldTag)
		}

		fields = append(fields, query)
	}

	return fields, nil
}

func buildNode(n *Node) (reflect.StructField, error) {
	var children []reflect.StructField
	var res reflect.StructField

	if len(n.scalars) > 0 {
		scalars, err := buildScalarQuery(n.scalars)
		if err != nil {
			return res, fmt.Errorf("error building scalar query at child node: %+v\nerr: %v", n.scalars, err)
		}
		children = append(children, scalars...)
	}

	if len(n.objects) > 0 {
		oq, err := buildObjectQuery(n.objects)
		if err != nil {
			return res, fmt.Errorf("error building object query at child node: %v", err)
		}

		children = append(children, oq...)
	}

	if len(n.unions) > 0 {
		uq, err := buildUnionQuery(n.unions)
		if err != nil {
			return res, fmt.Errorf("error building union query at child node: %v", err)
		}
		children = append(children, uq...)
	}

	res = reflect.StructField{
		Name: n.name,
		Type: reflect.StructOf(children),
	}

	if n.name == TypeNodeList {
		res.Type = reflect.SliceOf(res.Type)
	}

	return res, nil
}

// buildTag builds the struct tag from items in Object.tag
func buildTag(tags tag, name string) string {
	if len(tags) == 0 {
		return ""
	}

	var res string
	for k, t := range tags {
		if reflect.TypeOf(t).Name() == "string" {
			res += k + ": \\\"" + t.(string) + "\\\", "
		} else {
			res += fmt.Sprintf("%s: %v, ", k, t)
		}
	}

	// remove comma and space from the end of the string
	res = res[:len(res)-2]
	name = strings.ToLower(name[0:1]) + name[1:]
	return fmt.Sprintf(`graphql:"%s(%s)"`, name, res)
}
