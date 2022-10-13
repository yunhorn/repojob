package storage

import "testing"

func Test_DBStorage(t *testing.T) {
	storage := &GithubIssueStorage{}
	storage.Init()
	storage.Save()
}
