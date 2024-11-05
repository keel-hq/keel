package auth

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"
)

var expirationDelta = time.Hour * 12

type AuthType int

const (
	AuthTypeUnknown AuthType = iota
	AuthTypeBasic
	AuthTypeToken
)

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// JWT
	Token    string
	AuthType AuthType `json:"-"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"-"`
}

type Authenticator interface {
	// indicates whether authentication is enabled
	Enabled() bool
	Authenticate(req *AuthRequest) (*AuthResponse, error)
	GenerateToken(u User) (*AuthResponse, error)
}

func New(opts *Opts) *DefaultAuthenticator {
	if len(opts.Secret) == 0 {
		opts.Secret = []byte(randStringRunes(23))
	}

	return &DefaultAuthenticator{
		opts:   opts,
		secret: opts.Secret,
	}
}

type Opts struct {
	// Basic auth
	Username string
	Password string

	// Secret used to sign JWT tokens
	Secret []byte
}

type DefaultAuthenticator struct {
	opts *Opts

	secret []byte
}

var (
	ErrUnauthorized = errors.New("unauthorized")
)

func (a *DefaultAuthenticator) Enabled() bool {
	return a.opts.Username != "" && a.opts.Password != ""
}

func (a *DefaultAuthenticator) Authenticate(req *AuthRequest) (*AuthResponse, error) {

	switch req.AuthType {
	case AuthTypeToken:
		user, err := a.parseToken(req.Token)
		if err != nil {
			return nil, err
		}
		return a.GenerateToken(*user)
	case AuthTypeBasic:
		// ok
	default:
		return nil, fmt.Errorf("unknown auth type")
	}

	if a.opts.Username == "" && a.opts.Password == "" {
		// if basic auth not set - authenticating as guest
		return a.GenerateToken(User{Username: "guest"})
	}

	if req.Username != a.opts.Username || req.Password != a.opts.Password {
		return nil, ErrUnauthorized
	}

	return a.GenerateToken(User{Username: req.Username})
}

type User struct {
	Username string
}

func (a *DefaultAuthenticator) GenerateToken(u User) (*AuthResponse, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"username": "admin",
		"exp":      time.Now().Add(expirationDelta).Unix(),
		"iat":      time.Now().Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(a.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token, error: %s, s: %s", err, string(a.secret))
	}

	return &AuthResponse{
		Token: tokenString,
	}, nil
}

func (a *DefaultAuthenticator) parseToken(tokenString string) (*User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		user := &User{}
		user.Username = parseString(claims, "username")
		if user.Username == "" {
			log.WithFields(log.Fields{
				"token": tokenString,
				"error": "token is missing account username field",
			}).Warn("authenticator: malformed token")
			return nil, fmt.Errorf("malformed token")
		}

		// returning
		return user, nil

	}
	return nil, fmt.Errorf("invalid token")

}

func parseString(meta map[string]interface{}, key string) string {
	val, ok := meta[key]
	if !ok {
		return ""
	}

	s, ok := val.(string)
	if ok {
		return s
	}

	return ""
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
