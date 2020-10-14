package query

import (
	"reflect"
	"testing"

	"github.com/shurcooL/githubv4"
)

func TestNewCompoundItem_SetName(t *testing.T) {
	tests := []struct {
		name    string
		opt     Option
		want    *CompoundItem
		wantErr bool
	}{
		{
			name:    "valid",
			opt:     SetName("PullRequest"),
			want:    &CompoundItem{name: "PullRequest", tag: make(tag, 1)},
			wantErr: false,
		},
		{
			name:    "valid/initial starts with lower case",
			opt:     SetName("pullRequest"),
			want:    &CompoundItem{name: "PullRequest", tag: make(tag, 1)},
			wantErr: false,
		},
		{
			name:    "invalid/empty",
			opt:     SetName("  "),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid/contains symbols",
			opt:     SetName("pull-request_test"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid/contains digits",
			opt:     SetName("pullrequest10"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid/contains space",
			opt:     SetName("pull request"),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCompoundItem(tt.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCompoundItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCompoundItem() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewCompoundItem_SetFirst(t *testing.T) {

	tests := []struct {
		name    string
		opts    []Option
		want    *CompoundItem
		wantErr bool
	}{
		{
			name: "valid",
			opts: []Option{SetFirst(10)},
			want: &CompoundItem{
				tag: tag{
					"first": 10,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid/value",
			opts:    []Option{SetFirst(-1)},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid/contains last",
			opts: []Option{
				SetLast(10),
				SetFirst(10),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCompoundItem(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCompoundItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCompoundItem() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewCompoundItem_SetLast(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		want    *CompoundItem
		wantErr bool
	}{
		{
			name: "valid",
			opts: []Option{SetLast(10)},
			want: &CompoundItem{
				tag: tag{
					"last": 10,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid/value",
			opts:    []Option{SetLast(-1)},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid/contains first",
			opts: []Option{
				SetFirst(10),
				SetLast(10),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCompoundItem(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCompoundItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCompoundItem() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewCompoundItem_SetBefore(t *testing.T) {
	tests := []struct {
		name    string
		opt     Option
		want    *CompoundItem
		wantErr bool
	}{
		{
			name: "valid",
			opt:  SetBefore("test"),
			want: &CompoundItem{
				tag: tag{"before": "test"},
			},
			wantErr: false,
		},
		{
			name:    "invalid",
			opt:     SetBefore("  "),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCompoundItem(tt.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCompoundItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCompoundItem() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewCompoundItem_SetAfter(t *testing.T) {
	tests := []struct {
		name    string
		opt     Option
		want    *CompoundItem
		wantErr bool
	}{
		{
			name: "valid",
			opt:  SetAfter("test"),
			want: &CompoundItem{
				tag: tag{"after": "test"},
			},
			wantErr: false,
		},
		{
			name:    "invalid",
			opt:     SetAfter("  "),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCompoundItem(tt.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCompoundItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCompoundItem() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewCompoundItem_SetSearchType(t *testing.T) {
	tests := []struct {
		name    string
		opt     Option
		want    *CompoundItem
		wantErr bool
	}{
		{
			name: "valid",
			opt:  SetSearchType("issue"),
			want: &CompoundItem{
				tag: tag{
					"type": githubv4.SearchTypeIssue,
				},
			},
			wantErr: false,
		},
		{
			name: "valid/white space",
			opt:  SetSearchType(" issue"),
			want: &CompoundItem{
				tag: tag{
					"type": githubv4.SearchTypeIssue,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid",
			opt:     SetSearchType("test"),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCompoundItem(tt.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCompoundItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCompoundItem() got = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestNewCompoundItem_multipleOptions(t *testing.T) {
	type args struct {
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *CompoundItem
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				opts: []Option{
					SetName("PullRequest"),
					SetFirst(10),
					SetSearchType("issue"),
					SetQuery("author:test is:pr is:OPEN archived:false"),
				},
			},
			want: &CompoundItem{
				name: "PullRequest",
				tag: tag{
					"first": 10,
					"type":  githubv4.SearchTypeIssue,
					"query": "\"author:test is:pr is:OPEN archived:false\"",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCompoundItem(tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCompoundItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCompoundItem() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCompoundItem_tagExists(t *testing.T) {
	tests := []struct {
		name string
		tag  tag
		key  string
		want bool
	}{
		{
			name: "key found",
			tag: tag{
				"first": 1,
				"after": 2,
				"name":  3,
			},
			key:  "first",
			want: true,
		},
		{
			name: "key not found",
			tag: tag{
				"last":  1,
				"after": 2,
				"name":  3,
			},
			key:  "first",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CompoundItem{
				name: "test",
				tag:  tt.tag,
			}
			if got := c.tagExists(tt.key); got != tt.want {
				t.Errorf("tagExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
