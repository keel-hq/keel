package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	request "github.com/golang-jwt/jwt/v4/request"
	"github.com/keel-hq/keel/pkg/auth"
	log "github.com/sirupsen/logrus"
)

func authHeadersMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	rw.Header().Set("Access-Control-Allow-Headers",
		"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	rw.Header().Set("Access-Control-Expose-Headers", "Authorization")
	rw.Header().Set("Access-Control-Request-Headers", "Authorization")

	next(rw, r)
}

func (s *TriggerServer) requireAdminAuthorization(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// rw.Header().Set("Access-Control-Expose-Headers", "Authorization")
		// rw.Header().Set("Access-Control-Request-Headers", "Authorization")
		//
		if r.Method == "OPTIONS" {
			rw.WriteHeader(200)
			return
		}

		if s.authMode == auth.ModeExternalProxy {
			username := strings.TrimSpace(r.Header.Get(s.authProxyUserHeader))
			if !validProxyIdentity(username) {
				log.WithField("header", s.authProxyUserHeader).Warn("external authentication proxy request is missing a valid identity header")
				http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			r = auth.SetAuthenticationDetails(r, &auth.User{Username: username})
			next(rw, r)
			return
		}

		username, password, ok := r.BasicAuth()
		if ok {
			resp, err := s.authenticator.Authenticate(&auth.AuthRequest{
				Username: username,
				Password: password,
				AuthType: auth.AuthTypeBasic,
			})

			if err != nil {
				// Never log the password (or the full authorization
				// header): a failed webhook request must not leak its
				// credentials into log output.
				log.WithFields(log.Fields{
					"error": err,
					"user":  username,
				}).Error("authentication failed")
				// rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			r = auth.SetAuthenticationDetails(r, &resp.User)

			next(rw, r)
			return
		}

		// authenticating via token

		resp, err := s.authenticator.Authenticate(&auth.AuthRequest{
			Token:    extractToken(r),
			AuthType: auth.AuthTypeToken,
		})

		if err != nil {
			// rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

			// Never log the token itself: it is the caller's credential.
			log.WithField("error", err).Warn("authentication by token failed")
			http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		r = auth.SetAuthenticationDetails(r, &resp.User)

		next(rw, r)
	}
}

func validProxyIdentity(username string) bool {
	if username == "" || len(username) > 256 {
		return false
	}
	for _, r := range username {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func extractToken(req *http.Request) string {
	ex := request.AuthorizationHeaderExtractor
	token, err := ex.ExtractToken(req)
	if err != nil {
		return ""
	}

	return token
}

// logoutGetHandler documents the legacy GET logout operation.
// @Summary Log out with GET
// @Description Returns an empty JSON object. This route exists only when the authenticator is enabled.
// @Tags Auth
// @ID logoutViaGet
// @Produce json
// @Security BasicAuth
// @Security BearerAuth
// @Success 200 {object} EmptyResponse
// @Failure 401 {string} string "Unauthorized"
// @Router /v1/auth/logout [get]
func (s *TriggerServer) logoutGetHandler(resp http.ResponseWriter, req *http.Request) {
	s.logoutHandler(resp, req)
}

// logoutPostHandler documents the POST logout operation.
// @Summary Log out
// @Description Returns an empty JSON object. This route exists only when the authenticator is enabled.
// @Tags Auth
// @ID logout
// @Produce json
// @Security BasicAuth
// @Security BearerAuth
// @Success 200 {object} EmptyResponse
// @Failure 401 {string} string "Unauthorized"
// @Router /v1/auth/logout [post]
func (s *TriggerServer) logoutPostHandler(resp http.ResponseWriter, req *http.Request) {
	s.logoutHandler(resp, req)
}

func (s *TriggerServer) logoutHandler(resp http.ResponseWriter, req *http.Request) {

	resp.WriteHeader(200)
	resp.Write([]byte(`{}`))
}

// LoginRequest is the JSON payload accepted by the login endpoint.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// EmptyResponse documents endpoints that return an empty JSON object.
type EmptyResponse struct{}

// loginHandler authenticates an administrator.
// @Summary Log in
// @Description Authenticates Basic credentials supplied as JSON. This route exists only when the authenticator is enabled.
// @Tags Auth
// @ID login
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login credentials"
// @Success 200 {object} auth.AuthResponse
// @Header 200 {string} Authorization "Bearer token"
// @Failure 400 {string} string "Malformed request"
// @Failure 401 {string} string "Incorrect username or password"
// @Router /v1/auth/login [post]
func (s *TriggerServer) loginHandler(resp http.ResponseWriter, req *http.Request) {

	var lr LoginRequest
	dec := json.NewDecoder(req.Body)
	defer req.Body.Close()

	err := dec.Decode(&lr)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(resp, "%s", err)
		return
	}

	authResp, err := s.authenticator.Authenticate(&auth.AuthRequest{
		Username: lr.Username,
		Password: lr.Password,
		AuthType: auth.AuthTypeBasic,
	})

	if err != nil {
		log.Warnf("auth failed for user '%s', error: %s", lr.Username, err)
		http.Error(resp, "username or password incorrect", 401)
		return
	}

	log.Infof("auth successful for user %s", lr.Username)

	resp.Header().Add("Access-Control-Expose-Headers", "Authorization")
	resp.Header().Add("Authorization", fmt.Sprintf("Bearer %s", authResp.Token))

	response(authResp, 200, nil, resp, req)
}

// refreshHandler refreshes an administrator token.
// @Summary Refresh authentication
// @Description Generates a new bearer token. This route exists only when the authenticator is enabled.
// @Tags Auth
// @ID refreshAuth
// @Produce json
// @Security BasicAuth
// @Security BearerAuth
// @Success 200 {object} auth.AuthResponse
// @Header 200 {string} Authorization "Refreshed bearer token"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Token generation failed"
// @Router /v1/auth/refresh [get]
func (s *TriggerServer) refreshHandler(resp http.ResponseWriter, req *http.Request) {
	user := auth.GetAccountFromCtx(req.Context())

	authResp, err := s.authenticator.GenerateToken(*user)
	if err != nil {
		response(nil, http.StatusOK, err, resp, req)
		return
	}

	// adding token to header
	resp.Header().Add("Access-Control-Expose-Headers", "Authorization")
	resp.Header().Add("Authorization", fmt.Sprintf("Bearer %s", authResp.Token))

	response(authResp, http.StatusOK, err, resp, req)
}
