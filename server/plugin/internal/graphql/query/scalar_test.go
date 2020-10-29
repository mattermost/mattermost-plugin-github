package query

import (
	"reflect"
	"testing"
)

func TestNewScalar(t *testing.T) {
	type args struct {
		name string
		kind string
	}
	tests := []struct {
		name    string
		args    args
		want    Scalar
		wantErr bool
	}{
		{
			name: "valid/transforms id to uppercase",
			args: args{
				name: "id",
				kind: "ID",
			},
			want: Scalar{
				name: "ID",
				kind: "ID",
			},
			wantErr: false,
		},
		{
			name: "invalid/empty name",
			args: args{
				name: " ",
				kind: "String",
			},
			want:    Scalar{},
			wantErr: true,
		},
		{
			name: "invalid/name contains non-alpha characters",
			args: args{
				name: "test10-test",
				kind: "String",
			},
			want:    Scalar{},
			wantErr: true,
		},
		{
			name: "invalid/empty kind",
			args: args{
				name: "body",
				kind: "",
			},
			want:    Scalar{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewScalar(tt.args.name, tt.args.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewScalar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewScalar() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewScalarGroup(t *testing.T) {
	sg := map[string]string{
		"ID":     "id",
		"Body":   "String",
		"Number": "Int",
	}

	got, err := NewScalarGroup(sg)
	if err != nil {
		t.Errorf("NewScalarGroup() error = %v", err)
		return
	}

	want := []Scalar{
		{
			name: "ID",
			kind: "ID",
		},
		{
			name: "Body",
			kind: "String",
		},
		{
			name: "Number",
			kind: "Int",
		},
	}

	if len(got) != len(want) {
		t.Errorf("NewScalarGroup() got = %v, want %v", got, want)
		return
	}

	var found bool
	for _, w := range want {
		for _, g := range got {
			if reflect.DeepEqual(w, g) {
				found = true
				break
			}
		}

		if found == false {
			t.Errorf("NewScalarGroup() got = %v, want %v", got, want)
			break
		} else {
			found = false
		}
	}
}
