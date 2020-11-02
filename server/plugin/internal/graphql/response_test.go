package graphql

import (
	"net/url"
	"reflect"
	"testing"
	"time"
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
