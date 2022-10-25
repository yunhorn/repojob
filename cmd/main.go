package main

import (
	"context"
	"log"
	"os"
	"strings"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/google/go-github/v47/github" // with go modules enabled (GO111MODULE=on or outside GOPATH)
	"github.com/yunhorn/repojob/pkg/storage"
	"golang.org/x/oauth2"
)

var (
	ghclient     *github.Client
	ghToken      = os.Getenv("GITHUB_TOKEN")
	owner        = os.Getenv("OWNER")
	repo         = os.Getenv("REPO")
	issueStorage *storage.GithubIssueStorage
)

func init() {
	if ghToken == "" || owner == "" || repo == "" {
		log.Fatal("please set ENV GITHUB_TOKEN|OWNER|REPO")
	}
	ghclient = getClient()

	db, err := badger.Open(badger.DefaultOptions("data/badger"))
	if err != nil {
		log.Fatal(err)
	}

	issueStorage = &storage.GithubIssueStorage{
		Db: db,
	}
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

	maxPage := 10
	page := 1
	for i := 0; i < maxPage; i++ {
		issues, resp, err := ghclient.Issues.ListByRepo(ctx, owner, repo, &github.IssueListByRepoOptions{
			State: "all",
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    page,
			},
		})
		if err != nil {
			panic(err)
		}
		jobForissues(ctx, owner, repo, issues)
		page = resp.NextPage
		if page == 0 {
			break
		}
	}
}

func jobForissues(ctx context.Context, owner, repo string, issues []*github.Issue) {
	log.Println("issue.len:", len(issues))
	for i := 0; i < len(issues); i++ {
		issue := issues[i]

		issueCache := issueStorage.Get(owner, repo, *issue.Number)
		if issueCache.UpdateAt == nil || issue.UpdatedAt.After(*issueCache.UpdateAt) {
			go func(owner, repo string, issue *github.Issue) {
				log.Println("update issue updateTime:", owner, repo, *issue.Number)
				issueStorage.Set(owner, repo, *issue.Number, &storage.IssueCache{
					UpdateAt: issue.UpdatedAt,
				})
			}(owner, repo, issue)
		} else {
			continue
		}

		comments, _, err := ghclient.Issues.ListComments(ctx, owner, repo, *issue.Number, &github.IssueListCommentsOptions{})
		if err != nil {
			panic(err)
		}
		// log.Println("comments.len:", *issue.Number, *issue.Title, len(comments))
		log.Println("comments.len:", *issue.UpdatedAt, *issue.Number, len(comments))
		if issue.GetBody() != "" {
			issueBody := &github.IssueComment{
				Body: issue.Body,
				User: issue.User,
			}
			comments = append(comments, issueBody)
		}

		if len(comments) > 0 {
			ops := findOperationFromCommenct(comments)
			for o := 0; o < len(ops); o++ {
				op := ops[o]
				log.Println("op:", op.Name, op.Action, *issue.Number, op.Labels, op.Assigners)
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
				} else if op.Name == "issueassign" {
					if op.Action == "assign" {
						assignIssue(ctx, owner, repo, *issue.Number, op.Assigners)
					}
					if op.Action == "unassign" {
						removeAssign(ctx, owner, repo, *issue.Number, op.Assigners)
					}
				}

			}
		}
	}
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
		user := c.GetUser()
		ros := CommandFromComment(c.GetBody(), user.GetLogin())
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
			if r.Name == "issueassign" {
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
			if r.Name == "issueassign" {
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

func CommandFromComment(comment, user string) []*RepoOperation {
	result := []*RepoOperation{}
	if !strings.Contains(comment, "/assign") && !strings.Contains(comment, "/kind ") && !strings.Contains(comment, "/remove-kind ") && !strings.Contains(comment, "/close") && !strings.Contains(comment, "/reopen") {
		return result
	}

	labels := []string{}
	removeLabels := []string{}
	strs := strings.Split(comment, "\n")
	for i := 0; i < len(strs); i++ {
		str := strs[i]
		if strings.Contains(str, "/kind") {
			label := strings.ReplaceAll(str, "/kind ", "kind/")
			label = strings.Replace(label, "\r", "", -1)
			label = strings.Replace(label, " ", "", -1)
			labels = append(labels, strings.Replace(label, "\r", "", -1))
		}
		if strings.Contains(str, "/remove-kind") {
			label := strings.ReplaceAll(str, "/remove-kind ", "kind/")
			label = strings.Replace(label, "\r", "", -1)
			label = strings.Replace(label, " ", "", -1)
			removeLabels = append(removeLabels, label)
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
		if strings.Contains(str, "/assign") {
			ro := &RepoOperation{}
			ro.Name = "issueassign"
			ro.Action = "assign"
			assignersStr := strings.ReplaceAll(str, "/assign", "")
			if assignersStr == "" {
				//TODO self
				assignersStr = user
			} else {
				assignersStr = strings.ReplaceAll(assignersStr, "@", "")
				assignersStr = strings.Replace(assignersStr, "\r", "", -1)
				assignersStr = strings.ReplaceAll(assignersStr, " ", "")
			}
			ro.Assigners = []string{assignersStr}
			result = append(result, ro)
		}
		if strings.Contains(str, "/unassign") {
			ro := &RepoOperation{}
			ro.Name = "issueassign"
			ro.Action = "unassign"
			assignersStr := strings.ReplaceAll(str, "/unassign ", "")
			if assignersStr == "" {
				//TODO self
				assignersStr = user
			} else {
				assignersStr = strings.ReplaceAll(assignersStr, "@", "")
				assignersStr = strings.Replace(assignersStr, "\r", "", -1)
				assignersStr = strings.ReplaceAll(assignersStr, " ", "")
			}
			ro.Assigners = []string{assignersStr}
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
	Assigners   []string
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

func assignIssue(ctx context.Context, owner, repo string, issueId int, assigners []string) {
	if len(assigners) > 0 {
		_, _, err := ghclient.Issues.AddAssignees(ctx, owner, repo, issueId, assigners)
		if err != nil {
			panic(err)
		}
	}
}

func removeAssign(ctx context.Context, owner, repo string, issueId int, unassigners []string) {
	if len(unassigners) > 0 {
		_, _, err := ghclient.Issues.RemoveAssignees(ctx, owner, repo, issueId, unassigners)
		if err != nil {
			panic(err)
		}
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
