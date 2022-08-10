package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/keel-hq/keel/approvals"
	b "github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/bot/slack"
	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/pkg/store/sql"
	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/types"
	testutil "github.com/keel-hq/keel/util/testing"
)

var botMessagesChannel chan *b.BotMessage
var approvalsRespCh chan *b.ApprovalResponse

func New(name, token, channel string,
	k8sImplementer kubernetes.Implementer,
	approvalsManager approvals.Manager) *slack.Bot {

	approvalsRespCh = make(chan *b.ApprovalResponse)
	botMessagesChannel = make(chan *b.BotMessage)

	slack := &slack.Bot{}
	b.RegisterBot(name, slack)
	b.Run(k8sImplementer, approvalsManager)
	return slack
}

func setupEnv() (*sql.SQLStore, func()) {
	dir, err := ioutil.TempDir("", "whstoretest")
	if err != nil {
		log.Fatal(err)
	}
	tmpfn := filepath.Join(dir, "gorm.db")
	// defer
	store, err := sql.New(sql.Opts{DatabaseType: "sqlite3", URI: tmpfn})
	if err != nil {
		log.Fatal(err)
	}

	teardown := func() {
		os.RemoveAll(dir) // clean up
	}

	return store, teardown
}

func main() {
	f8s := &testutil.FakeK8sImplementer{}
	token := os.Getenv(constants.EnvSlackToken)

	store, teardown := setupEnv()
	defer teardown()

	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	New("keel", token, "approvals", f8s, am)
	defer b.Stop()

	time.Sleep(1 * time.Second)

	err := am.Create(&types.Approval{
		Identifier:     "k8s/project/repo:1.2.3",
		VotesRequired:  1,
		CurrentVersion: "2.3.4",
		NewVersion:     "3.4.5",
		Event: &types.Event{
			Repository: types.Repository{
				Name: "project/repo",
				Tag:  "2.3.4",
			},
		},
	})

	if err != nil {
		fmt.Printf("unexpected error while creating : %s", err)
	}

	time.Sleep(1 * time.Second)

}
