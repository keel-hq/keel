package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"

	"github.com/rusenask/keel/provider"

	log "github.com/Sirupsen/logrus"
)

// Opts - http server options
type Opts struct {
	Port int

	// available providers
	Providers map[string]provider.Provider
}

// TriggerServer - webhook trigger & healthcheck server
type TriggerServer struct {
	providers map[string]provider.Provider
	port      int
	server    *http.Server
	router    *mux.Router
}

// NewTriggerServer - create new HTTP trigger based server
func NewTriggerServer(opts *Opts) *TriggerServer {
	return &TriggerServer{
		port:      opts.Port,
		providers: opts.Providers,
		router:    mux.NewRouter(),
	}
}

// Start - start server
func (s *TriggerServer) Start() error {

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
	// health endpoint for k8s to be happy
	mux.HandleFunc("/healthz", s.healthHandler).Methods("GET", "OPTIONS")
	// native webhooks handler
	mux.HandleFunc("/v1/native", s.nativeHandler).Methods("POST", "OPTIONS")
}

func (s *TriggerServer) healthHandler(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)
}
