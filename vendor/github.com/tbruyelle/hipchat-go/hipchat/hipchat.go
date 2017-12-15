// Package hipchat provides a client for using the HipChat API v2.
package hipchat

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	defaultBaseURL = "https://api.hipchat.com/v2/"
)

// HTTPClient is an interface that allows overriding the http behavior
// by providing custom http clients
type HTTPClient interface {
	Do(req *http.Request) (res *http.Response, err error)
}

// LimitData contains the latest Rate Limit or Flood Control data sent with every API call response.
//
// Limit is the number of API calls per period of time
// Remaining is the current number of API calls that can be done before the ResetTime
// ResetTime is the UTC time in Unix epoch format for when the full Limit of API calls will be restored.
type LimitData struct {
	Limit     int
	Remaining int
	ResetTime int
}

// Client manages the communication with the HipChat API.
//
// LatestFloodControl contains the response from the latest API call's response headers X-Floodcontrol-{Limit, Remaining, ResetTime}
// LatestRateLimit contains the response from the latest API call's response headers X-Ratelimit-{Limit, Remaining, ResetTime}
// Room gives access to the /room part of the API.
// User gives access to the /user part of the API.
// Emoticon gives access to the /emoticon part of the API.
type Client struct {
	authToken          string
	BaseURL            *url.URL
	client             HTTPClient
	LatestFloodControl LimitData
	LatestRateLimit    LimitData
	Room               *RoomService
	User               *UserService
	Emoticon           *EmoticonService
}

// Links represents the HipChat default links.
type Links struct {
	Self string `json:"self"`
}

// PageLinks represents the HipChat page links.
type PageLinks struct {
	Links
	Prev string `json:"prev"`
	Next string `json:"next"`
}

// ID represents a HipChat id.
// Use a separate struct because it can be a string or a int.
type ID struct {
	ID string `json:"id"`
}

// ListOptions specifies the optional parameters to various List methods that
// support pagination.
//
// For paginated results, StartIndex represents the first page to display.
// For paginated results, MaxResults reprensents the number of items per page.  Default value is 100.  Maximum value is 1000.
type ListOptions struct {
	StartIndex int `url:"start-index,omitempty"`
	MaxResults int `url:"max-results,omitempty"`
}

// ExpandOptions specifies which Hipchat collections to automatically expand.
// This functionality is primarily used to reduce the total time to receive the data.
// It also reduces the sheer number of API calls from 1+N, to 1.
//
// cf:  https://developer.atlassian.com/hipchat/guide/hipchat-rest-api/api-title-expansion
type ExpandOptions struct {
	Expand string `url:"expand,omitempty"`
}

// Color is set of hard-coded string values for the HipChat API for notifications.
// cf: https://www.hipchat.com/docs/apiv2/method/send_room_notification
type Color string

const (
	// ColorYellow is the color yellow
	ColorYellow Color = "yellow"
	// ColorGreen is the color green
	ColorGreen Color = "green"
	// ColorRed is the color red
	ColorRed Color = "red"
	// ColorPurple is the color purple
	ColorPurple Color = "purple"
	// ColorGray is the color gray
	ColorGray Color = "gray"
	// ColorRandom is the random "surprise me!" color
	ColorRandom Color = "random"
)

// AuthTest can be set to true to test an auth token.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/auth#auth_test
var AuthTest = false

// AuthTestResponse will contain the server response of any
// API calls if AuthTest=true.
var AuthTestResponse = map[string]interface{}{}

// RetryOnRateLimit can be set to true to automatically retry the API call until it succeeds,
// subject to the RateLimitRetryPolicy settings.  This behavior is only active when the API
// call returns 429 (StatusTooManyRequests).
var RetryOnRateLimit = false

// RetryPolicy defines a RetryPolicy.
//
// MaxRetries is the maximum number of attempts to make before returning an error
// MinDelay is the initial delay between attempts.  This value is multiplied by the current attempt number.
// MaxDelay is the largest delay between attempts.
// JitterDelay is the amount of random jitter to add to the delay.
// JitterBias is the amount of jitter to remove from the delay.
//
// The use of Jitter avoids inadvertant and undesirable synchronization of network
// operations between otherwise unrelated clients.
// cf: https://brooker.co.za/blog/2015/03/21/backoff.html and https://www.awsarchitectureblog.com/2015/03/backoff.html
//
// Using the values of JitterDelay = 250 milliseconds and a JitterBias of negative 125 milliseconds,
// would result in a uniformly distributed Jitter between -125 and +125 milliseconds, centered
// around the current trial Delay (between MinDelay and MaxDelay).
//
//
type RetryPolicy struct {
	MaxRetries  int
	MinDelay    time.Duration
	MaxDelay    time.Duration
	JitterDelay time.Duration
	JitterBias  time.Duration
}

// NoRateLimitRetryPolicy defines the "never retry an API call" policy's values.
var NoRateLimitRetryPolicy = RetryPolicy{0, 1 * time.Second, 1 * time.Second, 500 * time.Millisecond, 0 * time.Millisecond}

// DefaultRateLimitRetryPolicy defines the "up to 300 times, 1 second apart, randomly adding an additional up-to-500 milliseconds of delay" policy.
var DefaultRateLimitRetryPolicy = RetryPolicy{300, 1 * time.Second, 1 * time.Second, 500 * time.Millisecond, 0 * time.Millisecond}

// RateLimitRetryPolicy can be set to a custom RetryPolicy's values,
// or to one of the two predefined ones: NoRateLimitRetryPolicy or DefaultRateLimitRetryPolicy
var RateLimitRetryPolicy = DefaultRateLimitRetryPolicy

// NewClient returns a new HipChat API client. You must provide a valid
// AuthToken retrieved from your HipChat account.
func NewClient(authToken string) *Client {
	baseURL, err := url.Parse(defaultBaseURL)
	if err != nil {
		panic(err)
	}

	c := &Client{
		authToken: authToken,
		BaseURL:   baseURL,
		client:    http.DefaultClient,
	}
	c.Room = &RoomService{client: c}
	c.User = &UserService{client: c}
	c.Emoticon = &EmoticonService{client: c}
	return c
}

// SetHTTPClient sets the http client for performing API requests.
// This method allows overriding the default http client with any
// implementation of the HTTPClient interface. It is typically used
// to have finer control of the http request.
// If a nil httpClient is provided, http.DefaultClient will be used.
func (c *Client) SetHTTPClient(httpClient HTTPClient) {
	if httpClient == nil {
		c.client = http.DefaultClient
	} else {
		c.client = httpClient
	}
}

// NewRequest creates an API request. This method can be used to performs
// API request not implemented in this library. Otherwise it should not be
// be used directly.
// Relative URLs should always be specified without a preceding slash.
func (c *Client) NewRequest(method, urlStr string, opt interface{}, body interface{}) (*http.Request, error) {
	rel, err := addOptions(urlStr, opt)
	if err != nil {
		return nil, err
	}

	if AuthTest {
		// Add the auth_test param
		values := rel.Query()
		values.Add("auth_test", strconv.FormatBool(AuthTest))
		rel.RawQuery = values.Encode()
	}

	u := c.BaseURL.ResolveReference(rel)

	buf := new(bytes.Buffer)
	if body != nil {
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+c.authToken)
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

// NewFileUploadRequest creates an API request to upload a file.
// This method manually formats the request as multipart/related with a single part
// of content-type application/json and a second part containing the file to be sent.
// Relative URLs should always be specified without a preceding slash.
func (c *Client) NewFileUploadRequest(method, urlStr string, v interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)

	shareFileReq, ok := v.(*ShareFileRequest)
	if !ok {
		return nil, errors.New("ShareFileRequest corrupted")
	}
	path := shareFileReq.Path
	message := shareFileReq.Message

	// Resolve home path
	if strings.HasPrefix(path, "~") {
		usr, _ := user.Current()
		path = strings.Replace(path, "~", usr.HomeDir, 1)
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	// Read file and encode to base 64
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	b64 := base64.StdEncoding.EncodeToString(file)
	contentType := mime.TypeByExtension(filepath.Ext(path))

	// Set proper filename
	filename := shareFileReq.Filename
	if filename == "" {
		filename = filepath.Base(path)
	} else if filepath.Ext(filename) != filepath.Ext(path) {
		filename = filepath.Base(filename) + filepath.Ext(path)
	}

	// Build request body
	body := "--hipfileboundary\n" +
		"Content-Type: application/json; charset=UTF-8\n" +
		"Content-Disposition: attachment; name=\"metadata\"\n\n" +
		"{\"message\": \"" + message + "\"}\n" +
		"--hipfileboundary\n" +
		"Content-Type: " + contentType + " charset=UTF-8\n" +
		"Content-Transfer-Encoding: base64\n" +
		"Content-Disposition: attachment; name=file; filename=" + filename + "\n\n" +
		b64 + "\n" +
		"--hipfileboundary\n"

	b := &bytes.Buffer{}
	_, err = b.Write([]byte(body))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, u.String(), b)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+c.authToken)
	req.Header.Add("Content-Type", "multipart/related; boundary=hipfileboundary")

	return req, err
}

// Do performs the request, the json received in the response is decoded
// and stored in the value pointed by v.
// Do can be used to perform the request created with NewRequest, which
// should be used only for API requests not implemented in this library.
func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	var policy = NoRateLimitRetryPolicy
	if RetryOnRateLimit {
		policy = RateLimitRetryPolicy
	}

	resp, err := c.doWithRetryPolicy(req, policy)
	if err != nil {
		return nil, err
	}

	if AuthTest {
		// If AuthTest is enabled, the reponse won't be the
		// one defined in the API endpoint.
		err = json.NewDecoder(resp.Body).Decode(&AuthTestResponse)
	} else {
		if c := resp.StatusCode; c < 200 || c > 299 {
			return resp, fmt.Errorf("Server returns status %d", c)
		}

		if v != nil {
			defer resp.Body.Close()
			if w, ok := v.(io.Writer); ok {
				_, err = io.Copy(w, resp.Body)
			} else {
				err = json.NewDecoder(resp.Body).Decode(v)
			}
		}
	}
	return resp, err
}

func (c *Client) doWithRetryPolicy(req *http.Request, policy RetryPolicy) (*http.Response, error) {
	currentTry := 0

	for willContinue(currentTry, policy) {
		currentTry = currentTry + 1
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		c.captureRateLimits(resp)
		if http.StatusTooManyRequests == resp.StatusCode {
			resp.Body.Close()
			if willContinue(currentTry, policy) {
				sleep(currentTry, policy)
			}
		} else {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("max retries exceeded (%d)", policy.MaxRetries)
}

func willContinue(currentTry int, policy RetryPolicy) bool {
	return currentTry <= policy.MaxRetries
}

func sleep(currentTry int, policy RetryPolicy) {
	jitter := time.Duration(rand.Int63n(2*int64(policy.JitterDelay))) - policy.JitterBias
	linearDelay := time.Duration(currentTry)*policy.MinDelay + jitter
	if linearDelay > policy.MaxDelay {
		linearDelay = policy.MaxDelay
	}
	time.Sleep(time.Duration(linearDelay))
}

func setIfPresent(src string, dest *int) {
	if len(src) > 0 {
		v, err := strconv.Atoi(src)
		if err != nil {
			*dest = v
		}
	}
}

func (c *Client) captureRateLimits(resp *http.Response) {
	// BY DESIGN:
	// if and only if the HTTP Response headers contain the header are the values updated.
	// The Floodcontrol limits are orthogonal to the API limits.
	// API Limits are consumed for each and every API call.
	// The default value for API limits are 500 (app token) or 100 (user token).
	// Flood Control limits are consumed only when a user message, room message, or room notification is sent.
	// The default value for Flood Control limits is 30 per minute per user token.
	setIfPresent(resp.Header.Get("X-Ratelimit-Limit"), &c.LatestRateLimit.Limit)
	setIfPresent(resp.Header.Get("X-Ratelimit-Remaining"), &c.LatestRateLimit.Remaining)
	setIfPresent(resp.Header.Get("X-Ratelimit-Reset"), &c.LatestRateLimit.ResetTime)
	setIfPresent(resp.Header.Get("X-Floodcontrol-Limit"), &c.LatestFloodControl.Limit)
	setIfPresent(resp.Header.Get("X-Floodcontrol-Remaining"), &c.LatestFloodControl.Remaining)
	setIfPresent(resp.Header.Get("X-Floodcontrol-Reset"), &c.LatestFloodControl.ResetTime)
}

// addOptions adds the parameters in opt as URL query parameters to s.  opt
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opt interface{}) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if opt == nil {
		return u, nil
	}

	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		// No query string to add
		return u, nil
	}

	qs, err := query.Values(opt)
	if err != nil {
		return nil, err
	}

	u.RawQuery = qs.Encode()
	return u, nil
}
