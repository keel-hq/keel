package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
)

type Opts struct {
	Port int

	// available providers
	Providers map[string]provider.Provider
}

// TriggerServer - webhook trigger
type TriggerServer struct {
	providers map[string]provider.Provider
	port      int
	server    *http.Server
	router    *mux.Router
}

func NewTriggerServer(opts *Opts) *TriggerServer {

	return &TriggerServer{
		port:      opts.Port,
		providers: opts.Providers,
	}
}

func (s *TriggerServer) Start() error {
	s.router = mux.NewRouter()

	s.registerRoutes(s.router)

	n := negroni.New(negroni.NewRecovery())
	n.UseHandler(s.router)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: n,
	}

	log.WithFields(log.Fields{
		"port": s.port,
	}).Info("webhook trigger server starting...")
	return s.server.ListenAndServe()
}

// Stop - stop webhook server
func (s *TriggerServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.server.Shutdown(ctx)

}

func (s *TriggerServer) registerRoutes(mux *mux.Router) {
	mux.HandleFunc("/healthz", s.healthHandler).Methods("GET", "OPTIONS")
	mux.HandleFunc("/native", s.nativeHandler).Methods("POST", "OPTIONS")
}

func (s *TriggerServer) healthHandler(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)
}

// nativeHandler - used to trigger event directly
func (s *TriggerServer) nativeHandler(resp http.ResponseWriter, req *http.Request) {
	event := types.Event{}
	if err := json.NewDecoder(req.Body).Decode(&event); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("failed to decode request")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	event.CreatedAt = time.Now()

	for _, p := range s.providers {
		err := p.Submit(event)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"provider": p.GetName(),
			}).Error("trigger.webhook: got error while submitting event to provider")
		}
	}

	resp.WriteHeader(http.StatusOK)
	return
}
