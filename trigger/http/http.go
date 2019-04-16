package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// Username and password are used for basic auth
	Username string
	Password string
}

// TriggerServer - webhook trigger & healthcheck server
type TriggerServer struct {
	providers        provider.Providers
	approvalsManager approvals.Manager
	port             int
	server           *http.Server
	router           *mux.Router

	// basic auth
	username string
	password string
}

// NewTriggerServer - create new HTTP trigger based server
func NewTriggerServer(opts *Opts) *TriggerServer {
	return &TriggerServer{
		port:             opts.Port,
		providers:        opts.Providers,
		approvalsManager: opts.ApprovalManager,
		router:           mux.NewRouter(),
		username:         opts.Username,
		password:         opts.Password,
	}
}

// Start - start server
func (s *TriggerServer) Start() error {

	s.registerRoutes(s.router)
	s.registerWebhookRoutes(s.router)

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

	if os.Getenv("DEBUG") == "true" {
		DebugHandler{}.AddRoutes(mux)
	}

	// health endpoint for k8s to be happy
	mux.HandleFunc("/healthz", s.healthHandler).Methods("GET", "OPTIONS")
	// version handler
	mux.HandleFunc("/version", s.versionHandler).Methods("GET", "OPTIONS")

	// approvals
	mux.HandleFunc("/v1/approvals", s.requireAdminAuthorization(s.approvalsHandler)).Methods("GET", "OPTIONS")
	// approving
	mux.HandleFunc("/v1/approvals", s.requireAdminAuthorization(s.approvalApproveHandler)).Methods("POST", "OPTIONS")

	mux.Handle("/metrics", promhttp.Handler())
}

func (s *TriggerServer) registerWebhookRoutes(mux *mux.Router) {
	mux.HandleFunc("/v1/webhooks/native", s.nativeHandler).Methods("POST", "OPTIONS")
	mux.HandleFunc("/v1/webhooks/dockerhub", s.dockerHubHandler).Methods("POST", "OPTIONS")
	mux.HandleFunc("/v1/webhooks/quay", s.quayHandler).Methods("POST", "OPTIONS")
	mux.HandleFunc("/v1/webhooks/azure", s.azureHandler).Methods("POST", "OPTIONS")

	// Docker registry notifications, used by Docker, Gitlab, Harbor
	// https://docs.docker.com/registry/notifications/
	//https://docs.gitlab.com/ee/administration/container_registry.html#configure-container-registry-notifications
	mux.HandleFunc("/v1/webhooks/registry", s.registryNotificationHandler).Methods("POST", "OPTIONS")
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

func (s *TriggerServer) requireAdminAuthorization(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		if s.username == "" && s.password == "" {
			next(rw, r)
			return
		}

		rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

		username, password, ok := r.BasicAuth()
		if ok && username == s.username && password == s.password {
			next(rw, r)
			return
		}

		http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
}
