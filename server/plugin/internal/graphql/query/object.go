package query

import (
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
)

type (
	// Object is the programmatic representation of object types in GraphQL
	// For more information about GraphQL object type see: https://graphql.org/learn/schema/#object-types-and-fields
	Object struct {
		name     string
		scalars  []Scalar
		objects  []Object
		node     *Node
		nodeList *Node
		unions   []Union
		tag      tag
	}

	// Option defines a function which sets items to Object.tag
	Option func(item *Object) error

	// tag items form the final query struct tag
	tag map[string]interface{}
)

// NewObject creates and returns a pointer to an Object
// When query is "search(last) {...}", then name should be "Search"
// When query is "... on PullRequest {...}", then name should be "PullRequest"
func NewObject(name string, opts ...Option) (*Object, error) {
	if err := validKey(name); err != nil {
		return nil, err
	}

	obj := &Object{
		name: strings.Title(strings.TrimSpace(name)),
		tag:  make(tag, 1),
	}

	for _, opt := range opts {
		err := opt(obj)
		if err != nil {
			return nil, err
		}
	}

	return obj, nil
}

// SetFirst adds the "first" tag to the tag list
func SetFirst(val int) Option {
	return func(item *Object) error {
		if err := greaterThan(val, 0); err != nil {
			return err
		}

		if item.tagExists("last") {
			return fmt.Errorf("cannot use 'first' and 'last' at the same time")
		}

		item.tag["first"] = val
		return nil
	}
}

// SetLast adds the "last" tag to the tag list
func SetLast(val int) Option {
	return func(item *Object) error {
		if err := greaterThan(val, 0); err != nil {
			return err
		}

		if item.tagExists("first") {
			return fmt.Errorf("cannot use 'first' and 'last' at the same time")
		}

		item.tag["last"] = val
		return nil
	}
}

// SetBefore adds the "before" tag to the tag list
func SetBefore(val string) Option {
	return func(item *Object) error {
		if err := strNotEmpty(val); err != nil {
			return err
		}

		item.tag["before"] = val
		return nil
	}
}

// SetAfter adds the "after" tag to the tag list
func SetAfter(val string) Option {
	return func(item *Object) error {
		if err := strNotEmpty(val); err != nil {
			return err
		}

		item.tag["after"] = val
		return nil
	}
}

// SetQuery adds the "query" tag to the tag list
func SetQuery(val string) Option {
	return func(item *Object) error {
		if err := strNotEmpty(val); err != nil {
			return err
		}

		item.tag["query"] = val
		return nil
	}
}

// SetSearchType adds the "type" tag to the tag list for search queries
// SearchType is an enum so val is checked and error is returned if an unexpected val is sent.
func SetSearchType(val string) Option {
	return func(item *Object) error {
		val = strings.ToUpper(strings.TrimSpace(val))
		var searchType githubv4.SearchType

		searchTypes := []githubv4.SearchType{
			githubv4.SearchTypeIssue,
			githubv4.SearchTypeUser,
			githubv4.SearchTypeRepository,
		}

		for _, v := range searchTypes {
			if val == string(v) {
				searchType = v
				break
			}
		}

		if searchType == "" {
			return fmt.Errorf("unexpected search type")
		}

		item.tag["type"] = searchType
		return nil
	}
}

func (o *Object) tagExists(key string) bool {
	for k := range o.tag {
		if k == key {
			return true
		}
	}

	return false
}

// AddScalar appends the given Scalar variable to its children
func (o *Object) AddScalar(scalar Scalar) {
	o.scalars = append(o.scalars, scalar)
}

// AddScalarGroup appends the given Scalar slice to its children
func (o *Object) AddScalarGroup(scalars []Scalar) {
	o.scalars = append(o.scalars, scalars...)
}

// AddObject appends the given Object variable to its children
func (o *Object) AddObject(obj *Object) {
	o.objects = append(o.objects, *obj)
}

// SetNode sets the node in to Object.node
// The given Node's name is checked in order to prevent setting a "node list" to Object.node
func (o *Object) SetNode(n *Node) error {
	if n.name == TypeNodeList {
		return fmt.Errorf("cannot set node list to node")
	}

	o.node = n
	return nil
}

// SetNodeList sets the give node to nodeList
// The given Node's name is checked in order to prevent setting a "node" to Object.nodeList
func (o *Object) SetNodeList(n *Node) error {
	if n.name == TypeNode {
		return fmt.Errorf("cannot set node to node list")
	}

	o.nodeList = n
	return nil
}

// AddUnion appends the given Union variable to its children
func (o *Object) AddUnion(u *Union) {
	o.unions = append(o.unions, *u)
}
