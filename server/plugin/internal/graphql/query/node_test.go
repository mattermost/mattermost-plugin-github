package query

import (
	"reflect"
	"testing"
)

func TestNewNode(t *testing.T) {
	want := &Node{name: TypeNode}
	if got := NewNode(); !reflect.DeepEqual(got, want) {
		t.Errorf("NewNode() = %+v, want %+v", got, want)
	}
}

func TestNewNodeList(t *testing.T) {
	want := &Node{name: TypeNodeList}
	if got := NewNodeList(); !reflect.DeepEqual(got, want) {
		t.Errorf("NewNode() = %+v, want %+v", got, want)
	}
}

func TestNode_AddScalar(t *testing.T) {
	s, _ := NewScalar("Body", "String")
	s2, _ := NewScalar("Title", "String")

	tests := []struct {
		name string
		arg  []Scalar
		node *Node
		want *Node
	}{
		{
			name: "type node",
			arg:  []Scalar{s, s2},
			node: NewNode(),
			want: &Node{
				name:    TypeNode,
				scalars: []Scalar{s, s2},
			},
		},
		{
			name: "type node list",
			arg:  []Scalar{s, s2},
			node: NewNodeList(),
			want: &Node{
				name:    TypeNodeList,
				scalars: []Scalar{s, s2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.node.AddScalar(s)
			tt.node.AddScalar(s2)

			if !reflect.DeepEqual(tt.node, tt.want) {
				t.Errorf("AddScalar() = %+v\nwant: %+v", tt.node, tt.want)
			}
		})
	}
}

func TestNode_AddObject(t *testing.T) {
	o, _ := NewObject(SetName("Search"))
	s, _ := NewScalar("Body", "String")
	o.AddScalar(s)

	tests := []struct {
		name string
		arg  *Object
		node *Node
		want *Node
	}{
		{
			name: "type node",
			arg:  o,
			node: NewNode(),
			want: &Node{
				name:    TypeNode,
				objects: []Object{*o},
			},
		},
		{
			name: "type node list",
			arg:  o,
			node: NewNodeList(),
			want: &Node{
				name:    TypeNodeList,
				objects: []Object{*o},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.node.AddObject(o)

			if !reflect.DeepEqual(tt.node, tt.want) {
				t.Errorf("AddObject() = %+v\nwant: %+v", tt.node, tt.want)
			}
		})
	}
}

func TestNode_AddUnion(t *testing.T) {
	u, _ := NewUnion("PullRequest")

	tests := []struct {
		name string
		arg  *Union
		node *Node
		want *Node
	}{
		{
			name: "type node",
			arg:  u,
			node: NewNode(),
			want: &Node{
				name:   TypeNode,
				unions: []Union{*u},
			},
		},
		{
			name: "type node list",
			arg:  u,
			node: NewNodeList(),
			want: &Node{
				name:   TypeNodeList,
				unions: []Union{*u},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.node.AddUnion(u)

			if !reflect.DeepEqual(tt.node, tt.want) {
				t.Errorf("AddUnion() = %+v\nwant: %+v", tt.node, tt.want)
			}
		})
	}
}
