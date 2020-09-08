package teams

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"fmt"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/extension/notification/teams"
)

func TestTrimLeftChar() {
	fmt.Printf("%q\n", "Hello, 世界")
    fmt.Printf("%q\n", teams.trimLeftChar(""))
    fmt.Printf("%q\n", teams.trimLeftChar("H"))
    fmt.Printf("%q\n", teams.trimLeftChar("世"))
    fmt.Printf("%q\n", teams.trimLeftChar("Hello"))
    fmt.Printf("%q\n", teams.trimLeftChar("世界"))
}

func TestTeamsRequest(t *testing.T) {
	currentTime := time.Now()
	handler := func(resp http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to parse body: %s", err)
		}

		bodyStr := string(body)

		if !strings.Contains(bodyStr, types.NotificationPreDeploymentUpdate.String()) {
			t.Errorf("missing deployment type")
		}

		if !strings.Contains(bodyStr, "debug") {
			t.Errorf("missing level")
		}

		if !strings.Contains(bodyStr, "update deployment") {
			t.Errorf("missing name")
		}
		if !strings.Contains(bodyStr, "message here") {
			t.Errorf("missing message")
		}

		t.Log(bodyStr)

	}

	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	s := &sender{
		webhook: ts.URL,
		client:   &http.Client{},
	}

	s.Send(types.EventNotification{
		Name:      "update deployment",
		Message:   "message here",
		CreatedAt: currentTime,
		Type:      types.NotificationPreDeploymentUpdate,
		Level:     types.LevelDebug,
	})
}
