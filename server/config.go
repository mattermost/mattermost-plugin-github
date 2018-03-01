package main

import "fmt"

type Configuration struct {
	GithubToken   string
	GithubOrg     string
	WebhookSecret string
}

func (c *Configuration) IsValid() error {
	if c.GithubToken == "" {
		return fmt.Errorf("Must have a github token")
	}

	if c.GithubOrg == "" {
		return fmt.Errorf("Must have a github Org")
	}

	return nil
}
