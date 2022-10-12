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

// func requestFromPage() {
// 	maxCount := 10
// 	haveNextPage := true
// 	current := 0
// 	issueResp := resp
// 	for i := 0; i < 100; i++ {
// 		if issueResp.NextPage == 0 {
// 			haveNextPage = false
// 		}
// 		if !haveNextPage {
// 			break
// 		}
// 		if current > maxCount {
// 			break
// 		}
// 		current++
// 		issues2, resp, err := ghclient.Issues.ListByRepo(ctx, owner, repo, &github.IssueListByRepoOptions{
// 			State: "all",
// 			ListOptions: github.ListOptions{
// 				PerPage: 500,
// 				Page:    issueResp.NextPage,
// 			},
// 		})
// 		if err != nil {
// 			panic(err)
// 		}
// 		issueResp = resp
// 		log.Println("issue2", len(issues2), issueResp.NextPage)
// 	}
// }

func main() {
	ctx := context.Background()
	issues, resp, err := ghclient.Issues.ListByRepo(ctx, owner, repo, &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		panic(err)
	}

	log.Println("issue.len:", len(issues), resp.NextPage)
	for i := 0; i < len(issues); i++ {
		issue := issues[i]
		if *issue.Number != 318 {
			continue
		}
		comments, _, err := ghclient.Issues.ListComments(ctx, owner, repo, *issue.Number, &github.IssueListCommentsOptions{})
		if err != nil {
			panic(err)
		}
		// log.Println("comments.len:", *issue.Number, *issue.Title, len(comments))
		log.Println("comments.len:", *issue.UpdatedAt, *issue.Number, len(comments))

		if len(comments) > 0 {
			ops := findOperationFromCommenct(comments)
			// ops := CommandFromComment(*comments[len(comments)-1].Body)
			for o := 0; o < len(ops); o++ {
				op := ops[o]
				// log.Println("op:", op.Name, op.Action, *issue.Number, op.Labels)
				if op.Name == "label" {
					if op.Action == "add" {
						labels := []string{}

						for _, label := range op.Labels {
							exist := false
							for b := 0; b < len(issue.Labels); b++ {
								if label == *issue.Labels[b].Name {
									exist = true
									break
								}
							}
							if !exist {
								labels = append(labels, label)
							}
						}

						addLabel(ctx, owner, repo, *issue.Number, labels)
					}
					if op.Action == "remove" {
						labels := []string{}

						for _, label := range op.Labels {
							exist := false
							for b := 0; b < len(issue.Labels); b++ {
								if label == *issue.Labels[b].Name {
									exist = true
									break
								}
							}
							if exist {
								labels = append(labels, label)
							}
						}
						removeLabel(ctx, owner, repo, *issue.Number, labels)
					}
				} else if op.Name == "issue" {
					if op.Action == "close" {
						if *issue.State != "closed" {
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

func removeMult(source, dest []string) []string {
	result := []string{}
	needRemoveLabels := []string{}
	for _, reallyAddLabel := range source {
		for _, removeLabel := range dest {
			if reallyAddLabel == removeLabel {
				needRemoveLabels = append(needRemoveLabels, removeLabel)
			}
		}
	}
	for _, label := range dest {
		needRemove := false
		for _, needRemoveLabel := range needRemoveLabels {
			if label == needRemoveLabel {
				needRemove = true
			}
		}
		if !needRemove {
			result = append(result, label)
		}
	}
	return result
}

func findOperationFromCommenct(comments []*github.IssueComment) []*RepoOperation {
	result := []*RepoOperation{}

	newRosMap := make(map[string]*RepoOperation)
	currentRosMap := make(map[string]*RepoOperation)
	for i := 0; i < len(comments); i++ {
		c := comments[i]
		ros := CommandFromComment(*c.Body)
		if len(ros) == 0 {
			continue
		}

		newRo := []*RepoOperation{}
		for j := 0; j < len(ros); j++ {
			r := ros[j]
			key := r.Name + "#" + r.Action
			if r.Name == "issue" {
				key = r.Name
			}
			if r.Name == "label" {
				if r.Action == "add" {
					needToRemoveLabelOp, ok := newRosMap["label#remove"]
					if ok {
						needToRemoveLabelOp.Labels = removeMult(r.Labels, needToRemoveLabelOp.Labels)
					}

				}
				if r.Action == "remove" {
					needToRemoveLabelOp, ok := newRosMap["label#add"]
					if ok {
						needToRemoveLabelOp.Labels = removeMult(r.Labels, needToRemoveLabelOp.Labels)
					}
				}
			}
			newRosMap[key] = r
		}

		for k := 0; k < len(result); k++ {
			r := result[k]
			key := r.Name + "#" + r.Action
			if r.Name == "issue" {
				key = r.Name
			}
			currentRosMap[key] = r
		}

		for key, val := range newRosMap {
			currentRosMap[key] = val
		}

		for _, v := range currentRosMap {
			newRo = append(newRo, v)
		}

		result = newRo
	}

	return result
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
		str := strs[i]
		if strings.Contains(str, "/kind") {
			label := strings.ReplaceAll(str, "/kind ", "")
			labels = append(labels, strings.Replace(label, "\r", "", -1))
		}
		if strings.Contains(str, "/remove-kind") {
			label := strings.ReplaceAll(str, "/remove-kind ", "")
			removeLabels = append(removeLabels, strings.Replace(label, "\r", "", -1))
		}
		if strings.Contains(str, "/close") {
			ro := &RepoOperation{}
			ro.Name = "issue"
			ro.Action = "close"
			result = append(result, ro)
		}
		if strings.Contains(str, "/reopen") {
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
	if len(labels) == 0 {
		return
	}
	log.Println("add label", owner, repo, issueId, labels)
	_, _, err := ghclient.Issues.AddLabelsToIssue(ctx, owner, repo, issueId, labels)
	if err != nil {
		log.Println("rebot add label comment failed!", err)
	}
}

func removeLabel(ctx context.Context, owner, repo string, issueId int, labels []string) {
	if len(labels) == 0 {
		return
	}
	log.Println("remove labels:", owner, repo, issueId, labels)
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
