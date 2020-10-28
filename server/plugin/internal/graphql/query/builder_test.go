package query

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shurcooL/githubv4"
)

func Test_buildScalarQuery(t *testing.T) {
	tests := []struct {
		name    string
		args    []Scalar
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid_1",
			args: []Scalar{
				{
					name: "IssueCount",
					kind: "Int",
				},
				{
					name: "Title",
					kind: "String",
				},
				{
					name: "ID",
					kind: "ID",
				},
				{
					name: "Test",
					kind: "Float",
				},
			},
			want: struct {
				IssueCount githubv4.Int
				Title      githubv4.String
				ID         githubv4.ID
				Test       githubv4.Float
			}{},
			wantErr: false,
		},
		{
			name: "valid_2",
			args: []Scalar{
				{
					name: "BooleanField",
					kind: "Boolean",
				},
				{
					name: "DateField",
					kind: "Date",
				},
				{
					name: "DateTimeField",
					kind: "DateTime",
				},
				{
					name: "HTMLField",
					kind: "HTML",
				},
				{
					name: "URIField",
					kind: "uri",
				},
			},
			want: struct {
				BooleanField  githubv4.Boolean
				DateField     githubv4.Date
				DateTimeField githubv4.DateTime
				HTMLField     githubv4.HTML
				URIField      githubv4.URI
			}{},
			wantErr: false,
		},
		{
			name: "invalid/unknown type",
			args: []Scalar{
				{
					name: "ID",
					kind: "invalidtype",
				},
				{
					name: "Title",
					kind: "String",
				},
			},
			want:    struct{}{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildScalarQuery(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildScalarQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			query := reflect.New(reflect.StructOf(got)).Elem().Interface()
			if !reflect.DeepEqual(query, tt.want) {
				t.Errorf("buildScalarQuery()\n got = %T\nwant = %T", query, tt.want)
			}
		})
	}
}

func Test_buildTag(t *testing.T) {
	type args struct {
		tags tag
		name string
	}
	tests := []struct {
		name         string
		args         args
		want         string
		wantMultiple []string
	}{

		{
			name: "empty string",
			args: args{name: "test"},
			want: "",
		},
		{
			name: "single tag",
			args: args{
				tags: tag{
					"first": 100,
				},
				name: "Reviews",
			},
			want: `graphql:"reviews(first: 100)"`,
		},
		{
			name: "single string tag",
			args: args{
				tags: tag{
					"query": "test: test value",
				},
				name: "Reviews",
			},
			want: `graphql:"reviews(query: \"test: test value\")"`,
		},
		{
			name: "multiple elements",
			args: args{
				tags: tag{
					"first": 100,
					"query": "test: test value",
					"type":  githubv4.SearchTypeIssue,
				},
				name: "ReviewRequests",
			},
			wantMultiple: []string{
				"first: 100",
				"query: \\\"test: test value\\\"",
				"type: " + string(githubv4.SearchTypeIssue),
			},
			want: `graphql:"reviewRequests(first: 100, query: \"test: test value\", type: ISSUE)"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTag(tt.args.tags, tt.args.name)

			if len(tt.args.tags) == 1 && got != tt.want {
				t.Errorf("buildTag() = %s\nwant: %s", got, tt.want)
				return
			}

			if len(tt.wantMultiple) > 0 {
				for _, v := range tt.wantMultiple {
					if !strings.Contains(got, v) {
						t.Errorf("tag not found: %s\n\nbuildTag() = %s, want (order not important): %s", v, got, tt.want)
					}
				}
			}
		})
	}
}

func Test_buildObjectQuery(t *testing.T) {
	tests := []struct {
		name    string
		obj     []Object
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid/has scalar",
			obj: []Object{
				{
					name: "Search",
					scalars: []Scalar{
						{name: "IssueCount", kind: "Int"},
						{name: "Title", kind: "String"},
						{name: "ID", kind: "ID"},
					},
				},
			},
			want: struct {
				Search struct {
					IssueCount githubv4.Int
					Title      githubv4.String
					ID         githubv4.ID
				}
			}{},
			wantErr: false,
		},
		{
			name: "valid/has scalar and tag",
			obj: []Object{
				{
					name:    "ReviewRequests",
					scalars: []Scalar{{name: "TotalCount", kind: "Int"}},
					tag:     tag{"first": 10},
				},
			},
			want: struct {
				ReviewRequests struct {
					TotalCount githubv4.Int
				} `graphql:"reviewRequests(first: 10)"`
			}{},
			wantErr: false,
		},
		{
			name: "valid/has nested object queries",
			obj: []Object{
				{
					name: "Enterprise",
					objects: []Object{
						{
							name:    "BillingInfo",
							scalars: []Scalar{{name: "AssetPacks", kind: "Int"}},
						},
						{
							name:    "UserAccounts",
							scalars: []Scalar{{name: "TotalCount", kind: "Int"}},
							objects: []Object{
								{
									name:    "PageInfo",
									scalars: []Scalar{{name: "HasNextPage", kind: "Boolean"}},
								},
							},
						},
					},
				},
			},
			want: struct {
				Enterprise struct {
					BillingInfo struct {
						AssetPacks githubv4.Int
					}
					UserAccounts struct {
						TotalCount githubv4.Int
						PageInfo   struct {
							HasNextPage githubv4.Boolean
						}
					}
				}
			}{},
			wantErr: false,
		},
		{
			name: "valid/has node",
			obj: []Object{
				{
					name: "Search",
					scalars: []Scalar{
						{name: "IssueCount", kind: "Int"},
						{name: "Title", kind: "String"},
					},
					nodeList: &Node{
						name: TypeNodeList,
						unions: []Union{
							{
								name:    "PullRequest",
								scalars: []Scalar{{"ID", "ID"}, {"Body", "String"}},
							},
						},
					},
				},
			},
			want: struct {
				Search struct {
					IssueCount githubv4.Int
					Title      githubv4.String
					Nodes      []struct {
						PullRequest struct {
							ID   githubv4.ID
							Body githubv4.String
						} `graphql:"... on PullRequest"`
					}
				}
			}{},
			wantErr: false,
		},
		{
			name: "valid/has union",
			obj: []Object{
				{
					name: "Search",
					unions: []Union{
						{
							name:    "PullRequest",
							scalars: []Scalar{{"ID", "ID"}, {"Body", "String"}},
						},
					},
				},
			},
			want: struct {
				Search struct {
					PullRequest struct {
						ID   githubv4.ID
						Body githubv4.String
					} `graphql:"... on PullRequest"`
				}
			}{},
			wantErr: false,
		},
		{
			name:    "invalid/no children",
			obj:     []Object{{name: "test"}},
			want:    struct{}{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildObjectQuery(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildObjectQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			query := reflect.New(reflect.StructOf(got)).Elem().Interface()
			if !reflect.DeepEqual(query, tt.want) {
				t.Errorf("buildObjectQuery()\n got = %T\nwant = %T", query, tt.want)
			}
		})
	}
}

func Test_buildUnionQuery(t *testing.T) {
	tests := []struct {
		name    string
		u       []Union
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid/single with scalar",
			u: []Union{
				{
					name:    "PullRequest",
					scalars: []Scalar{{"ID", "ID"}, {"Body", "String"}},
				},
			},
			want: struct {
				PullRequest struct {
					ID   githubv4.ID
					Body githubv4.String
				} `graphql:"... on PullRequest"`
			}{},
			wantErr: false,
		},
		{
			name: "valid/single with object",
			u: []Union{
				{
					name:    "PullRequest",
					scalars: []Scalar{{"ID", "ID"}, {"Body", "String"}},
					objects: []Object{
						{
							name: "Repository",
							scalars: []Scalar{
								{name: "Name", kind: "String"},
								{name: "NameWithOwner", kind: "String"},
							},
						},
					},
				},
			},
			want: struct {
				PullRequest struct {
					ID         githubv4.ID
					Body       githubv4.String
					Repository struct {
						Name          githubv4.String
						NameWithOwner githubv4.String
					}
				} `graphql:"... on PullRequest"`
			}{},
			wantErr: false,
		},
		{
			name: "valid/multiple",
			u: []Union{
				{
					name:    "PullRequest",
					scalars: []Scalar{{"ID", "ID"}, {"Body", "String"}},
				},
				{
					name:    "App",
					scalars: []Scalar{{"ID", "ID"}, {"Name", "String"}},
				},
			},
			want: struct {
				PullRequest struct {
					ID   githubv4.ID
					Body githubv4.String
				} `graphql:"... on PullRequest"`
				App struct {
					ID   githubv4.ID
					Name githubv4.String
				} `graphql:"... on App"`
			}{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildUnionQuery(tt.u)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildUnionQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			query := reflect.New(reflect.StructOf(got)).Elem().Interface()
			if !reflect.DeepEqual(query, tt.want) {
				t.Errorf("buildUnionQuery()\n got = %T\nwant = %T", query, tt.want)
			}
		})
	}
}

func Test_buildNode(t *testing.T) {
	tests := []struct {
		name    string
		n       *Node
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid/Node",
			n: &Node{
				name:    TypeNode,
				scalars: []Scalar{{"ID", "ID"}, {"Body", "String"}},
				objects: []Object{
					{
						name: "Search",
						scalars: []Scalar{
							{name: "IssueCount", kind: "Int"},
							{name: "Title", kind: "String"},
						},
					},
				},
				unions: []Union{
					{
						name:    "PullRequest",
						scalars: []Scalar{{"ID", "ID"}, {"Body", "String"}},
					},
				},
			},
			want: struct {
				Node struct {
					ID     githubv4.ID
					Body   githubv4.String
					Search struct {
						IssueCount githubv4.Int
						Title      githubv4.String
					}
					PullRequest struct {
						ID   githubv4.ID
						Body githubv4.String
					} `graphql:"... on PullRequest"`
				}
			}{},
			wantErr: false,
		},
		{
			name: "valid/NodeList",
			n: &Node{
				name:    TypeNodeList,
				scalars: []Scalar{{"ID", "ID"}, {"Body", "String"}},
				objects: nil,
				unions: []Union{
					{
						name:    "PullRequest",
						scalars: []Scalar{{"ID", "ID"}, {"Body", "String"}},
					},
				},
			},
			want: struct {
				Nodes []struct {
					ID          githubv4.ID
					Body        githubv4.String
					PullRequest struct {
						ID   githubv4.ID
						Body githubv4.String
					} `graphql:"... on PullRequest"`
				}
			}{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildNode(tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			query := reflect.New(reflect.StructOf([]reflect.StructField{got})).Elem().Interface()
			if !reflect.DeepEqual(query, tt.want) {
				t.Errorf("buildNode()\n got = %T\nwant = %T", query, tt.want)
			}
		})
	}
}

func TestBuilder_Build(t *testing.T) {
	root := &Object{
		name:    "Search",
		scalars: []Scalar{{"IssueCount", "Int"}},
		nodeList: &Node{
			name: TypeNodeList,
			unions: []Union{
				{
					name: "PullRequest",
					scalars: []Scalar{
						{name: "ID", kind: "ID"},
						{name: "Body", kind: "String"},
						{name: "Mergeable", kind: "MergeableState"},
						{name: "Number", kind: "Int"},
						{name: "ReviewDecision", kind: "PullRequestReviewDecision"},
						{name: "State", kind: "PullRequestState"},
						{name: "Title", kind: "String"},
						{name: "URL", kind: "URI"},
					},
					objects: []Object{
						{
							name: "Repository",
							scalars: []Scalar{
								{name: "Name", kind: "String"},
								{name: "NameWithOwner", kind: "String"},
							},
						},
						{
							name:    "Reviews",
							scalars: []Scalar{{"TotalCount", "Int"}},
							nodeList: &Node{
								name:    TypeNodeList,
								scalars: []Scalar{{"State", "PullRequestState"}},
							},
							tag: tag{"first": 10},
						},
						{
							name:    "ReviewRequests",
							scalars: []Scalar{{"TotalCount", "Int"}},
							nodeList: &Node{
								name: TypeNodeList,
								objects: []Object{
									{
										name: "RequestedReviewer",
										unions: []Union{
											{
												name:    "User",
												scalars: []Scalar{{"Login", "String"}},
											},
										},
									},
								},
							},
							tag: tag{"first": 10},
						},
					},
				},
			},
		},
		tag: tag{
			"query": "author:test is:pr is:closed archived:false",
		},
	}
	var want struct {
		Search struct {
			IssueCount githubv4.Int
			Nodes      []struct {
				PullRequest struct {
					ID             githubv4.ID
					Body           githubv4.String
					Mergeable      githubv4.MergeableState
					Number         githubv4.Int
					ReviewDecision githubv4.PullRequestReviewDecision
					State          githubv4.PullRequestState
					Title          githubv4.String
					URL            githubv4.URI
					Repository     struct {
						Name          githubv4.String
						NameWithOwner githubv4.String
					}
					Reviews struct {
						TotalCount githubv4.Int
						Nodes      []struct {
							State githubv4.PullRequestState
						}
					} `graphql:"reviews(first: 10)"`
					ReviewRequests struct {
						TotalCount githubv4.Int
						Nodes      []struct {
							RequestedReviewer struct {
								User struct {
									Login githubv4.String
								} `graphql:"... on User"`
							}
						}
					} `graphql:"reviewRequests(first: 10)"`
				} `graphql:"... on PullRequest"`
			}
		} `graphql:"search(query: \"author:test is:pr is:closed archived:false\")"`
	}

	b := Builder{root: root}

	got, err := b.Build()
	if err != nil {
		t.Errorf("Build() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Build()\n got = %T\nwant = %T", got, want)
	}
}
