package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/pkg/auth"
	"github.com/keel-hq/keel/pkg/store"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/provider/kubernetes"
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

	Authenticator auth.Authenticator

	GRC *k8s.GenericResourceCache

	KubernetesClient kubernetes.Implementer

	Store store.Store

	UIDir string

	AuthenticatedWebhooks bool
}

// TriggerServer - webhook trigger & healthcheck server
type TriggerServer struct {
	grc              *k8s.GenericResourceCache
	kubernetesClient kubernetes.Implementer

	providers        provider.Providers
	approvalsManager approvals.Manager
	port             int
	server           *http.Server
	router           *mux.Router

	store         store.Store
	authenticator auth.Authenticator

	uiDir string

	authenticatedWebhooks bool
}

// NewTriggerServer - create new HTTP trigger based server
func NewTriggerServer(opts *Opts) *TriggerServer {
	return &TriggerServer{
		port:                  opts.Port,
		grc:                   opts.GRC,
		kubernetesClient:      opts.KubernetesClient,
		providers:             opts.Providers,
		approvalsManager:      opts.ApprovalManager,
		router:                mux.NewRouter(),
		authenticator:         opts.Authenticator,
		store:                 opts.Store,
		uiDir:                 opts.UIDir,
		authenticatedWebhooks: opts.AuthenticatedWebhooks,
	}
}

// Start - start server
func (s *TriggerServer) Start() error {

	s.registerRoutes(s.router)

	n := negroni.New(negroni.NewRecovery())
	n.Use(negroni.HandlerFunc(corsHeadersMiddleware))
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

	s.registerWebhookRoutes(mux)

	// health endpoint for k8s to be happy
	mux.HandleFunc("/healthz", s.healthHandler).Methods("GET", "OPTIONS")
	// version handler
	mux.HandleFunc("/version", s.versionHandler).Methods("GET", "OPTIONS")

	mux.Handle("/metrics", promhttp.Handler())

	if s.authenticator.Enabled() {
		log.Info("authentication enabled, setting up admin HTTP handlers")
		// auth
		mux.HandleFunc("/v1/auth/login", s.loginHandler).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/auth/info", s.requireAdminAuthorization(s.userInfoHandler)).Methods("GET", "OPTIONS")
		mux.HandleFunc("/v1/auth/user", s.requireAdminAuthorization(s.userInfoHandler)).Methods("GET", "OPTIONS")
		mux.HandleFunc("/v1/auth/logout", s.requireAdminAuthorization(s.logoutHandler)).Methods("POST", "GET", "OPTIONS")
		mux.HandleFunc("/v1/auth/refresh", s.requireAdminAuthorization(s.refreshHandler)).Methods("GET", "OPTIONS")

		// approvals
		mux.HandleFunc("/v1/approvals", s.requireAdminAuthorization(s.approvalsHandler)).Methods("GET", "OPTIONS")
		// approving/rejecting
		mux.HandleFunc("/v1/approvals", s.requireAdminAuthorization(s.approvalApproveHandler)).Methods("POST", "OPTIONS")
		// updating required approvals count
		mux.HandleFunc("/v1/approvals", s.requireAdminAuthorization(s.approvalSetHandler)).Methods("PUT", "OPTIONS")

		// available resources
		mux.HandleFunc("/v1/resources", s.requireAdminAuthorization(s.resourcesHandler)).Methods("GET", "OPTIONS")

		mux.HandleFunc("/v1/policies", s.requireAdminAuthorization(s.policyUpdateHandler)).Methods("PUT", "OPTIONS")

		// tracked images
		mux.HandleFunc("/v1/tracked", s.requireAdminAuthorization(s.trackedHandler)).Methods("GET", "OPTIONS")
		mux.HandleFunc("/v1/tracked", s.requireAdminAuthorization(s.trackSetHandler)).Methods("PUT", "OPTIONS")

		// status
		mux.HandleFunc("/v1/audit", s.requireAdminAuthorization(s.adminAuditLogHandler)).Methods("GET", "OPTIONS")
		mux.HandleFunc("/v1/stats", s.requireAdminAuthorization(s.statsHandler)).Methods("GET", "OPTIONS")

		if s.uiDir != "" {
			// Serve static assets directly.
			mux.PathPrefix("/css/").Handler(http.FileServer(http.Dir(s.uiDir)))
			mux.PathPrefix("/assets/").Handler(http.FileServer(http.Dir(s.uiDir)))
			mux.PathPrefix("/js/").Handler(http.FileServer(http.Dir(s.uiDir)))
			mux.PathPrefix("/img/").Handler(http.FileServer(http.Dir(s.uiDir)))
			mux.PathPrefix("/loading/").Handler(http.FileServer(http.Dir(s.uiDir)))

			mux.PathPrefix("/").HandlerFunc(indexHandler(s.uiDir))
		}
	} else {
		log.Info("authentication is not enabled, admin HTTP handlers are not initialized")
	}

}

func (s *TriggerServer) registerWebhookRoutes(mux *mux.Router) {

	if s.authenticatedWebhooks {
		mux.HandleFunc("/v1/webhooks/native", s.requireAdminAuthorization(s.nativeHandler)).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/dockerhub", s.requireAdminAuthorization(s.dockerHubHandler)).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/jfrog", s.requireAdminAuthorization(s.jfrogHandler)).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/quay", s.requireAdminAuthorization(s.quayHandler)).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/azure", s.requireAdminAuthorization(s.azureHandler)).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/github", s.requireAdminAuthorization(s.githubHandler)).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/harbor", s.requireAdminAuthorization(s.harborHandler)).Methods("POST", "OPTIONS")

		// Docker registry notifications, used by Docker, Gitlab, Harbor
		// https://docs.docker.com/registry/notifications/
		//https://docs.gitlab.com/ee/administration/container_registry.html#configure-container-registry-notifications
		mux.HandleFunc("/v1/webhooks/registry", s.registryNotificationHandler).Methods("POST", "OPTIONS")
	} else {
		mux.HandleFunc("/v1/webhooks/native", s.nativeHandler).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/dockerhub", s.dockerHubHandler).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/jfrog", s.jfrogHandler).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/quay", s.quayHandler).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/azure", s.azureHandler).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/github", s.githubHandler).Methods("POST", "OPTIONS")
		mux.HandleFunc("/v1/webhooks/harbor", s.harborHandler).Methods("POST", "OPTIONS")

		// Docker registry notifications, used by Docker, Gitlab, Harbor
		// https://docs.docker.com/registry/notifications/
		//https://docs.gitlab.com/ee/administration/container_registry.html#configure-container-registry-notifications
		mux.HandleFunc("/v1/webhooks/registry", s.registryNotificationHandler).Methods("POST", "OPTIONS")
	}
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

func response(obj interface{}, statusCode int, err error, resp http.ResponseWriter, req *http.Request) {
	// Check for an error

	if err != nil {

		code := 500
		errMsg := err.Error()
		if strings.Contains(errMsg, "Permission denied") {
			code = 403
		}
		resp.WriteHeader(code)
		resp.Write([]byte(err.Error()))
		return
	}

	// Write out the JSON object
	if obj != nil {

		resp.Header().Set("Content-Type", "application/json")
		resp.WriteHeader(statusCode)

		// Set up the pipe to write data directly into the Reader.
		pr, pw := io.Pipe()

		// Write JSON-encoded data to the Writer end of the pipe.
		// Write in a separate concurrent goroutine, and remember
		// to Close the PipeWriter, to signal to the paired PipeReader
		// that we’re done writing.
		go func() {
			pw.CloseWithError(json.NewEncoder(pw).Encode(obj))
		}()

		io.Copy(resp, pr)

		// encoding/json library has a specific bug(feature) to turn empty slices into json null object,
		// let's make an empty array instead
		// resp.Write(buf)
	}
}

// corsHeadersMiddleware - cors middleware
func corsHeadersMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	rw.Header().Set("Access-Control-Allow-Headers",
		"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	rw.Header().Set("Access-Control-Expose-Headers", "Authorization")
	rw.Header().Set("Access-Control-Request-Headers", "Authorization")

	if r.Method == "OPTIONS" {
		rw.WriteHeader(200)
		return
	}

	next(rw, r)
}

type UserInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Username      string `json:"username"`
	Avatar        string `json:"avatar"`
	Status        int    `json:"status"`
	LastLoginIP   string `json:"last_login_ip"`
	LastLoginTime int64  `json:"last_login_time"`
	RoleID        string `json:"role_id"`
}

func (s *TriggerServer) userInfoHandler(resp http.ResponseWriter, req *http.Request) {

	user := auth.GetAccountFromCtx(req.Context())

	ui := UserInfo{
		ID:            "1",
		Name:          user.Username,
		Avatar:        "",
		Status:        1,
		LastLoginIP:   "",
		LastLoginTime: time.Now().Unix(),
		RoleID:        "admin",
	}

	response(&ui, 200, nil, resp, req)
}

type APIResponse struct {
	Status string `json:"status"`
}

func indexHandler(uiDir string) func(w http.ResponseWriter, r *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, uiDir+"/index.html")
	}

	return http.HandlerFunc(fn)
}
