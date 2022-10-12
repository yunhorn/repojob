package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v47/github" // with go modules enabled (GO111MODULE=on or outside GOPATH)
	"golang.org/x/oauth2"
)

var (
	ghclient *github.Client
	ghToken  = os.Getenv("GITHUB_TOKEN")
	owner    = os.Getenv("OWNER")
	repo     = os.Getenv("REPO")
)

func init() {
	if ghToken == "" || owner == "" || repo == "" {
		log.Fatal("please set ENV GITHUB_TOKEN|OWNER|REPO")
	}
	ghclient = getClient()
}

func main() {
	ctx := context.Background()
	issues, _, err := ghclient.Issues.ListByRepo(ctx, owner, repo, &github.IssueListByRepoOptions{
		State: "all",
	})
	if err != nil {
		panic(err)
	}
	log.Println("issue.len:", len(issues))
	for i := 0; i < len(issues); i++ {
		issue := issues[i]
		if *issue.Number != 318 {
			continue
		}
		comments, _, err := ghclient.Issues.ListComments(ctx, owner, repo, *issue.Number, &github.IssueListCommentsOptions{})
		if err != nil {
			panic(err)
		}
		log.Println("comments.len:", *issue.Number, *issue.Title, len(comments))

		for j := 0; j < len(comments); j++ {
			// log.Println(*comments[j].Body)
		}
		if len(comments) > 0 {
			ops := CommandFromComment(*comments[len(comments)-1].Body)
			for o := 0; o < len(ops); o++ {
				op := ops[o]
				log.Println("op:", op.Name, op.Action)
				if op.Name == "label" {
					if op.Action == "add" {
						addLabel(ctx, owner, repo, *issue.Number, op.Labels)
					}
					if op.Action == "remove" {
						labels := []string{}
						for a := 0; a < len(issue.Labels); a++ {
							exist := false
							for b := 0; b < len(op.Labels); b++ {
								if op.Labels[b] == *issue.Labels[a].Name {
									exist = true
									break
								}
							}
							if exist {
								labels = append(labels, *issue.Labels[a].Name)
							}
						}
						removeLabel(ctx, owner, repo, *issue.Number, labels)
					}
				} else if op.Name == "issue" {
					if op.Action == "close" {
						if *issue.State != "close" {
							closeIssue(ctx, owner, repo, *issue.Number)
						}
					}
					if op.Action == "reopen" {
						if *issue.State != "open" {
							reopenIssue(ctx, owner, repo, *issue.Number)
						}
					}
				}
			}
		}
	}

	//TODO selector comment of have /kind {label}  /close /reopen /remove-kind
	//TODO save  comment count of issue,issues[i].Comments
}

func CommandFromComment(comment string) []*RepoOperation {
	result := []*RepoOperation{}
	if !strings.Contains(comment, "/kind ") && !strings.Contains(comment, "/remove-kind ") && !strings.Contains(comment, "/close") && !strings.Contains(comment, "/reopen") {
		return result
	}

	labels := []string{}
	removeLabels := []string{}
	strs := strings.Split(comment, "\n")
	for i := 0; i < len(strs); i++ {
		if strings.Contains(strs[i], "/kind") {
			label := strings.ReplaceAll(strs[i], "/kind ", "")
			log.Println(label)
			labels = append(labels, label)
		}
		if strings.Contains(strs[i], "/remove-kind") {
			label := strings.ReplaceAll(strs[i], "/remove-kind ", "")
			removeLabels = append(labels, label)
		}
		if strings.Contains(strs[i], "/close") {
			ro := &RepoOperation{}
			ro.Name = "issue"
			ro.Action = "close"
			result = append(result, ro)
		}
		if strings.Contains(strs[i], "/reopen") {
			ro := &RepoOperation{}
			ro.Name = "issue"
			ro.Action = "reopen"
			result = append(result, ro)
		}
	}
	if len(labels) > 0 {
		ro := &RepoOperation{}
		ro.Labels = labels
		ro.Name = "label"
		ro.Action = "add"
		result = append(result, ro)
	}
	if len(removeLabels) > 0 {
		ro := &RepoOperation{}
		ro.Labels = removeLabels
		ro.Name = "label"
		ro.Action = "remove"
		result = append(result, ro)
	}
	return result
}

type RepoOperation struct {
	Name        string //  label,issue
	Action      string // add-label remove-label open-issue close-issue
	Labels      []string
	IssueNumber int
	Reply       string
}

func addLabel(ctx context.Context, owner, repo string, issueId int, labels []string) {
	_, _, err := ghclient.Issues.AddLabelsToIssue(ctx, owner, repo, issueId, labels)
	if err != nil {
		log.Println("rebot add label comment failed!", err)
	}
}

func removeLabel(ctx context.Context, owner, repo string, issueId int, labels []string) {
	if len(labels) == 0 {
		return
	}
	log.Println("remove labels:", owner, repo, labels)
	for i := 0; i < len(labels); i++ {
		_, err := ghclient.Issues.RemoveLabelForIssue(ctx, owner, repo, issueId, labels[i])
		if err != nil {
			log.Println("rebot add label comment failed!", err)
		}
	}

}

func reopenIssue(ctx context.Context, owner, repo string, issueId int) {
	log.Println("reopen issue:", owner, repo, issueId)
	state := "open"
	_, _, err := ghclient.Issues.Edit(ctx, owner, repo, issueId, &github.IssueRequest{
		State: &state,
	})
	if err != nil {
		panic(err)
	}
}

func closeIssue(ctx context.Context, owner, repo string, issueId int) {
	log.Println("issue.close:", owner, repo, issueId)
	state := "closed"
	_, _, err := ghclient.Issues.Edit(ctx, owner, repo, issueId, &github.IssueRequest{
		State: &state,
	})
	if err != nil {
		panic(err)
	}
}

func getClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)

	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return client
}
