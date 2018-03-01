package main

import "fmt"

type Configuration struct {
	GithubToken string
}

func (c *Configuration) IsValid() error {
	if c.GithubToken == "" {
		return fmt.Errorf("Must have a github token")
	}

	return nil
}
