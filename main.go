package main

import (
	"context"
	"log"

	"github.com/google/go-github/v47/github" // with go modules enabled (GO111MODULE=on or outside GOPATH)
	"golang.org/x/oauth2"
)

func main() {
	ctx := context.Background()
	c := getClient()
	issues, _, err := c.Issues.ListByRepo(ctx, "yunhorn", "{repo}", &github.IssueListByRepoOptions{})
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(issues); i++ {
		log.Println(*issues[i].Number)
	}
	comments, _, err := c.Issues.ListComments(ctx, "yunhorn", "{repo}", *issues[0].Number, &github.IssueListCommentsOptions{})
	if err != nil {
		panic(err)
	}
	log.Println("comments.len:", len(comments))

	log.Println("issue.len:", len(issues))
	for i := 0; i < len(comments); i++ {
		log.Println(*comments[i].Body)
	}
	//TODO selector comment of have /kind {label}
	//TODO add label for some issue
}

func getClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "{GITHUB_TOKEN}"},
	)

	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return client
}
