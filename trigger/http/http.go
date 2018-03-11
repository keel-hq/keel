package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/version"

	log "github.com/sirupsen/logrus"
)

// Opts - http server options
type Opts struct {
	Port int

	// available providers
	Providers provider.Providers

	ApprovalManager approvals.Manager
}

// TriggerServer - webhook trigger & healthcheck server
type TriggerServer struct {
	providers        provider.Providers
	approvalsManager approvals.Manager
	port             int
	server           *http.Server
	router           *mux.Router
}

// NewTriggerServer - create new HTTP trigger based server
func NewTriggerServer(opts *Opts) *TriggerServer {
	return &TriggerServer{
		port:             opts.Port,
		providers:        opts.Providers,
		approvalsManager: opts.ApprovalManager,
		router:           mux.NewRouter(),
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

func getID(req *http.Request) string {
	return mux.Vars(req)["id"]
}

func (s *TriggerServer) registerRoutes(mux *mux.Router) {
	// health endpoint for k8s to be happy
	mux.HandleFunc("/healthz", s.healthHandler).Methods("GET", "OPTIONS")
	// version handler
	mux.HandleFunc("/version", s.versionHandler).Methods("GET", "OPTIONS")

	// approvals
	mux.HandleFunc("/v1/approvals", s.approvalsHandler).Methods("GET", "OPTIONS")
	// approving
	mux.HandleFunc("/v1/approvals", s.approvalApproveHandler).Methods("POST", "OPTIONS")

	// native webhooks handler
	mux.HandleFunc("/v1/webhooks/native", s.nativeHandler).Methods("POST", "OPTIONS")

	// dockerhub webhooks handler
	mux.HandleFunc("/v1/webhooks/dockerhub", s.dockerHubHandler).Methods("POST", "OPTIONS")
	mux.HandleFunc("/v1/webhooks/quay", s.quayHandler).Methods("POST", "OPTIONS")
}

func (s *TriggerServer) healthHandler(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)
}

func (s *TriggerServer) versionHandler(resp http.ResponseWriter, req *http.Request) {
	v := version.GetKeelVersion()

	encoded, err := json.Marshal(v)
	if err != nil {
		log.WithError(err).Error("trigger.http: failed to marshal version")
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(encoded)
}

func (s *TriggerServer) trigger(event types.Event) error {
	return s.providers.Submit(event)
}
