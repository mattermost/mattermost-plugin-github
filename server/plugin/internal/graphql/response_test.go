package graphql

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

func Test_convertToMap(t *testing.T) {
	uri, _ := url.Parse("https://mattermost.com")
	dt, _ := time.Parse(time.RFC3339, "2019-10-09T10:09:15Z")

	tests := []struct {
		name       string
		query      interface{}
		wantResult Response
		wantErr    bool
	}{
		{
			name: "valid/value",
			query: struct {
				Title      string
				IssueCount int
			}{
				Title:      "Test",
				IssueCount: 123,
			},
			wantResult: Response{
				"Title":      "Test",
				"IssueCount": 123,
			},
			wantErr: false,
		},
		{
			name: "valid/object types",
			query: &struct {
				URL      *url.URL
				DateTime time.Time
			}{
				URL:      uri,
				DateTime: dt,
			},
			wantResult: Response{
				"URL":      "https://mattermost.com",
				"DateTime": dt.Format("2006-01-02, 15:04:05"),
			},
			wantErr: false,
		},
		{
			name: "valid/ptr",
			query: &struct {
				Title      string
				IssueCount int
			}{
				Title:      "Test",
				IssueCount: 123,
			},
			wantResult: Response{
				"Title":      "Test",
				"IssueCount": 123,
			},
			wantErr: false,
		},
		{
			name: "valid/nested struct",
			query: &struct {
				Title      string
				IssueCount int
				Repository struct {
					Name string
				}
			}{
				Title:      "Test",
				IssueCount: 123,
				Repository: struct {
					Name string
				}{
					Name: "test",
				},
			},
			wantResult: Response{
				"Title":      "Test",
				"IssueCount": 123,
				"Repository": Response{
					"Name": "test",
				},
			},
			wantErr: false,
		},
		{
			name: "valid/nested struct slice",
			query: &struct {
				Title      string
				IssueCount int
				Repository struct {
					Name string
				}
				Nodes []struct {
					PullRequest struct {
						ID   int
						Body string
					}
				}
			}{
				Title:      "Test",
				IssueCount: 123,
				Repository: struct {
					Name string
				}{
					Name: "test",
				},
				Nodes: []struct {
					PullRequest struct {
						ID   int
						Body string
					}
				}{
					{
						PullRequest: struct {
							ID   int
							Body string
						}{ID: 1, Body: "pr body 1"},
					},
					{
						PullRequest: struct {
							ID   int
							Body string
						}{ID: 2, Body: "pr body 2"},
					},
				},
			},
			wantResult: Response{
				"Title":      "Test",
				"IssueCount": 123,
				"Repository": Response{
					"Name": "test",
				},
				"Nodes": []Response{
					{
						"PullRequest": Response{
							"ID":   1,
							"Body": "pr body 1",
						},
					},
					{
						"PullRequest": Response{
							"ID":   2,
							"Body": "pr body 2",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "invalid/empty parameter",
			query:      nil,
			wantResult: nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToResponse(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantResult) {
				t.Errorf("convertToResponse()\n got = %v\nwant = %v", got, tt.wantResult)
			}
		})
	}
}

func TestResponse_GetString(t *testing.T) {
	r := Response{"test": "test value"}
	want := "test value"
	if got := r.GetString("test"); got != want {
		t.Errorf("GetString() = %v, want %v", got, want)
	}

	r = Response{"test": githubv4.PullRequestStateMerged}
	want = "MERGED"
	if got := r.GetString("test"); got != want {
		t.Errorf("GetString() = %v, want %v", got, want)
	}

	r = Response{"test": 123}
	want = ""
	if got := r.GetString("test"); got != want {
		t.Errorf("GetString() = %v, want %v", got, want)
	}
}

func TestResponse_GetInt64(t *testing.T) {
	var want int64
	r := Response{"test": "test value"}
	want = 0
	if got := r.GetInt64("test"); got != want {
		t.Errorf("GetInt64() = %v, want %v", got, want)
	}

	r = Response{"test": 123}
	want = 123
	if got := r.GetInt64("test"); got != want {
		t.Errorf("GetInt64() = %v, want %v", got, want)
	}
}

func TestResponse_GetBool(t *testing.T) {
	r := Response{"test": true}
	want := true
	if got := r.GetBool("test"); got != want {
		t.Errorf("GetBool() = %v, want %v", got, want)
	}

	r = Response{"test": 123}
	want = false
	if got := r.GetBool("test"); got != want {
		t.Errorf("GetBool() = %v, want %v", got, want)
	}
}

func TestResponse_GetFloat64(t *testing.T) {
	var want float64
	r := Response{"test": 2.1234}
	want = 2.1234
	if got := r.GetFloat64("test"); got != want {
		t.Errorf("GetFloat64() = %v, want %v", got, want)
	}

	r = Response{"test": *githubv4.NewFloat(1.2345)}
	want = 1.2345
	if got := r.GetFloat64("test"); got != want {
		t.Errorf("GetFloat64() = %v, want %v", got, want)
	}

	r = Response{"test": 123}
	want = 0
	if got := r.GetFloat64("test"); got != want {
		t.Errorf("GetFloat64() = %v, want %v", got, want)
	}
}

func TestResponse_GetResponseObject(t *testing.T) {
	r := Response{"test": Response{"tst2": 1}}
	want := Response{"tst2": 1}
	if got := r.GetResponseObject("test"); !reflect.DeepEqual(got, want) {
		t.Errorf("GetResponseObject() = %v, want %v", got, want)
	}

	r = Response{"test": 123}
	want = nil
	if got := r.GetResponseObject("test"); !reflect.DeepEqual(got, want) {
		t.Errorf("GetResponseObject() = %v, want %v", got, want)
	}
}
