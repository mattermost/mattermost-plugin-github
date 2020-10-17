package query

import (
	"reflect"
	"testing"
)

func TestNode_AddScalar(t *testing.T) {
	s, _ := NewScalar("Body", "String")
	s2, _ := NewScalar("Title", "String")

	want := &Node{
		name:    "Nodes",
		scalars: []Scalar{s, s2},
	}

	got := NewNode()
	got.AddScalar(s)
	got.AddScalar(s2)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddScalar() = %+v\nwant: %+v", got, want)
	}
}

func TestNode_AddObject(t *testing.T) {
	o, _ := NewObject(SetName("Search"))
	s, _ := NewScalar("Body", "String")
	o.AddScalar(s)

	want := &Node{
		name:    "Nodes",
		objects: []Object{*o},
	}

	got := NewNode()
	got.AddObject(o)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddObject() = %+v\nwant: %+v", got, want)
	}
}
