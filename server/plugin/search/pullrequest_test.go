package search

import (
	"flag"
	"log"
	"testing"

	"github.com/mattermost/mattermost-plugin-github/server/plugin/internal/graphql"
	"golang.org/x/oauth2"
)

var accessToken string
var username string

func init() {
	flag.StringVar(&accessToken, "token", "", "Github user access token")
	flag.StringVar(&accessToken, "username", "", "Github username")
}

func TestGetPRDetail(t *testing.T) {
	if accessToken == "" || username == ""{
		t.Skipf("user access token or username not provided, skipping test")
	}

	gc := graphql.NewClient(oauth2.Token{AccessToken: accessToken}, username, "", "")

	res, err := GetPRDetail(gc)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	log.Printf("%+v", res)
}
