package query

import (
	"reflect"
	"testing"
)

func TestNewUnion(t *testing.T) {
	got, err := NewUnion("PullRequest")
	if err != nil {
		t.Errorf("NewUnion() error = %v", err)
		return
	}

	want := &Union{
		name: "PullRequest",
		tag: tag{
			TagKeyUnion: "... on PullRequest",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewUnion() got = %+v, want %+v", got, want)
	}
}

func TestUnion_SetNode(t *testing.T) {
	got, _ := NewUnion("PullRequest")

	// Check wrong type (node/node list) error
	if err := got.SetNode(NewNodeList()); err == nil {
		t.Errorf("SetNode() expected error, got nil")
		return
	}

	s, _ := NewScalar("Body", "String")
	n := NewNode()
	n.AddScalar(s)

	if err := got.SetNode(n); err != nil {
		t.Errorf("SetNode() err = %v", err)
		return
	}

	want := &Union{
		name: "PullRequest",
		node: n,
		tag:  tag{TagKeyUnion: "... on PullRequest"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("SetNode() = %+v\nwant: %+v", got, want)
	}
}

func TestUnion_AddObject(t *testing.T) {
	o, _ := NewObject(SetName("Search"), SetFirst(100))
	want := &Union{
		name:    "PullRequest",
		objects: []Object{*o},
		tag:     tag{TagKeyUnion: "... on PullRequest"},
	}

	got, _ := NewUnion("PullRequest")
	got.AddObject(o)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddObject() = %+v\nwant: %+v", got, want)
	}
}

func TestUnion_AddScalar(t *testing.T) {
	s, _ := NewScalar("Body", "String")
	s2, _ := NewScalar("Title", "String")

	want := &Union{
		name:    "PullRequest",
		scalars: []Scalar{s, s2},
		tag:     tag{TagKeyUnion: "... on PullRequest"},
	}

	got, _ := NewUnion("PullRequest")
	got.AddScalar(s)
	got.AddScalar(s2)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AddScalar() = %+v\nwant: %+v", got, want)
	}
}

func TestUnion_SetNodeList(t *testing.T) {
	got, _ := NewUnion("PullRequest")

	// Check wrong type (node/node list) error
	if err := got.SetNodeList(NewNode()); err == nil {
		t.Errorf("SetNodeList() expected error, got nil")
		return
	}

	s, _ := NewScalar("Body", "String")
	n := NewNodeList()
	n.AddScalar(s)

	if err := got.SetNodeList(n); err != nil {
		t.Errorf("SetNodeList() err = %v", err)
		return
	}

	want := &Union{
		name:     "PullRequest",
		nodeList: n,
		tag:      tag{TagKeyUnion: "... on PullRequest"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("SetNodeList() = %+v\nwant: %+v", got, want)
	}
}
