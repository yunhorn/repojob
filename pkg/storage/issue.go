package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/dgraph-io/badger/v3"
)

type IssueStorage interface {
	ScanALLed(owner, repo string) bool
	Set(owner, repo string, issueNumber int, issueCache *IssueCache) error
	Get(owner, repo string, issueNumber int) *IssueCache
}

type DBStorage interface {
	Init() error
	Save() error
}

type IssueCache struct {
	UpdateAt *time.Time
}

type GithubIssueStorage struct {
	Db *badger.DB
}

func (gh *GithubIssueStorage) Init() error {
	return nil
}

func (gh *GithubIssueStorage) Save() error {
	// repo, err := remote.NewRepository("ghcr.io/yunhorn/repojobdata")
	// if err != nil {
	// 	panic(err)
	// }
	// ctx := context.Background()

	// generateManifest := func(config ocispec.Descriptor, layers ...ocispec.Descriptor) ([]byte, error) {
	// 	content := ocispec.Manifest{
	// 		Config:    config,
	// 		Layers:    layers,
	// 		Versioned: specs.Versioned{SchemaVersion: 2},
	// 	}
	// 	return json.Marshal(content)
	// }
	// // 1. assemble descriptors and manifest
	// layerBlob := []byte("Hello layer")
	// layerDesc := content.NewDescriptorFromBytes(ocispec.MediaTypeImageLayer, layerBlob)
	// configBlob := []byte("Hello config")
	// configDesc := content.NewDescriptorFromBytes(ocispec.MediaTypeImageConfig, configBlob)
	// manifestBlob, err := generateManifest(configDesc, layerDesc)
	// if err != nil {
	// 	panic(err)
	// }
	// manifestDesc := content.NewDescriptorFromBytes(ocispec.MediaTypeImageManifest, manifestBlob)

	// // 2. push and tag
	// err = repo.Push(ctx, layerDesc, bytes.NewReader(layerBlob))
	// if err != nil {
	// 	panic(err)
	// }
	// err = repo.Push(ctx, configDesc, bytes.NewReader(configBlob))
	// if err != nil {
	// 	panic(err)
	// }
	// err = repo.PushReference(ctx, manifestDesc, bytes.NewReader(manifestBlob), "test")
	// if err != nil {
	// 	panic(err)
	// }

	fmt.Println("Succeed")
	return nil
}

func (gh *GithubIssueStorage) ScanALLed(owner, repo string) bool {
	return false
}
func (gh *GithubIssueStorage) Set(owner, repo string, issueNumber int, issueCache *IssueCache) error {
	return gh.Db.Update(func(txn *badger.Txn) error {
		key := owner + "/" + repo + "/" + strconv.Itoa(issueNumber)
		data, err := json.Marshal(issueCache)
		if err != nil {
			return err
		}

		e := badger.NewEntry([]byte(key), data).WithTTL(time.Hour)
		err = txn.SetEntry(e)
		return err
	})
}
func (gh *GithubIssueStorage) Get(owner, repo string, issueNumber int) *IssueCache {
	cacheValue := []byte{}
	key := owner + "/" + repo + "/" + strconv.Itoa(issueNumber)
	err := gh.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		var valNot, valCopy []byte
		err = item.Value(func(val []byte) error {
			valCopy = append([]byte{}, val...)
			valNot = val
			return nil
		})
		if err != nil {
			return err
		}

		// DO NOT access val here. It is the most common cause of bugs.
		if valNot != nil {
			// fmt.Printf("NEVER do this. %s\n", valNot)
		}

		// You must copy it to use it outside item.Value(...).
		// fmt.Printf("The answer is: %s\n", valCopy)

		// Alternatively, you could also use item.ValueCopy().
		valCopy, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		// fmt.Printf("The answer is: %s\n", valCopy)
		cacheValue = valCopy

		return nil
	})

	cache := &IssueCache{}
	if err != nil {
		log.Println("data.to.json.failed!", err)
		return cache
	}

	err = json.Unmarshal(cacheValue, cache)
	if err != nil {
		log.Println("data.to.json.failed!", err)
	}
	return cache
}
