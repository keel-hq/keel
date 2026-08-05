package mattermost

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keel-hq/keel/types"
)

func TestMattermostResponseStatuses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		redirect   bool
		wantErr    bool
	}{
		{name: "200 OK", statusCode: http.StatusOK},
		{name: "201 Created", statusCode: http.StatusCreated},
		{name: "202 Accepted", statusCode: http.StatusAccepted},
		{name: "204 No Content", statusCode: http.StatusNoContent},
		{name: "299 other 2xx", statusCode: 299},
		{name: "302 redirect", statusCode: http.StatusFound, redirect: true, wantErr: true},
		{name: "400 client error", statusCode: http.StatusBadRequest, wantErr: true},
		{name: "500 server error", statusCode: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if tt.redirect {
					http.Redirect(w, r, "/accepted", tt.statusCode)
					return
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			s := &sender{
				endpoint: server.URL,
				name:     "keel",
				client:   &http.Client{CheckRedirect: rejectRedirect},
			}

			err := s.Send(types.EventNotification{})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), "expected 2xx") {
				t.Errorf("Send() error = %q, want expected status diagnostic", err)
			}
			if got := requests.Load(); got != 1 {
				t.Errorf("request count = %d, want 1", got)
			}
		})
	}
}
