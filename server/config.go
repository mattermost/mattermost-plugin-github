package main

import "fmt"

type Configuration struct {
	GithubToken   string
	GithubOrg     string
	WebhookSecret string
	Username      string
}

func (c *Configuration) IsValid() error {
	if c.GithubToken == "" {
		return fmt.Errorf("Must have a github token")
	}

	if c.GithubOrg == "" {
		return fmt.Errorf("Must have a github Org")
	}

	if c.Username == "" {
		return fmt.Errorf("Need a username to make posts as.")
	}

	return nil
}
