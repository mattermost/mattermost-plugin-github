package query

import (
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
)

type (
	Object struct {
		name    string
		scalars []Scalar
		objects []Object
		node    *Node
		unions  []Union
		tag     tag
	}

	Option func(item *Object) error

	tag map[string]interface{}
)

func NewObject(opts ...Option) (*Object, error) {
	obj := &Object{
		tag: make(tag, 1),
	}

	for _, opt := range opts {
		err := opt(obj)
		if err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func SetName(val string) Option {
	return func(item *Object) error {
		if err := validKey(val); err != nil {
			return err
		}
		item.name = strings.Title(strings.TrimSpace(val))
		return nil
	}
}

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

func SetBefore(val string) Option {
	return func(item *Object) error {
		if err := strNotEmpty(val); err != nil {
			return err
		}

		item.tag["before"] = val
		return nil
	}
}

func SetAfter(val string) Option {
	return func(item *Object) error {
		if err := strNotEmpty(val); err != nil {
			return err
		}

		item.tag["after"] = val
		return nil
	}
}

func SetQuery(val string) Option {
	return func(item *Object) error {
		if err := strNotEmpty(val); err != nil {
			return err
		}

		item.tag["query"] = "\"" + val + "\""
		return nil
	}
}

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
	for k, _ := range o.tag {
		if k == key {
			return true
		}
	}

	return false
}

func (o *Object) AddScalar(scalar Scalar) {
	o.scalars = append(o.scalars, scalar)
}

func (o *Object) AddObject(obj *Object) {
	o.objects = append(o.objects, *obj)
}

func (o *Object) SetNode(n *Node) {
	o.node = n
}

func (o *Object) AddUnion(u *Union) {
	o.unions = append(o.unions, *u)
}
