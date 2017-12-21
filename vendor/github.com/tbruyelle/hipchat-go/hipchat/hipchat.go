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
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

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

// Client manages the communication with the HipChat API.
type Client struct {
	authToken string
	BaseURL   *url.URL
	client    HTTPClient
	// Room gives access to the /room part of the API.
	Room *RoomService
	// User gives access to the /user part of the API.
	User *UserService
	// Emoticon gives access to the /emoticon part of the API.
	Emoticon *EmoticonService
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

// ListOptions  specifies the optional parameters to various List methods that
// support pagination.
type ListOptions struct {
	// For paginated results, represents the first page to display.
	StartIndex int `url:"start-index,omitempty"`
	// For paginated results, reprensents the number of items per page.
	MaxResults int `url:"max-results,omitempty"`
}

type Color string

const (
	ColorYellow Color = "yellow"
	ColorGreen  Color = "green"
	ColorRed    Color = "red"
	ColorPurple Color = "purple"
	ColorGray   Color = "gray"
	ColorRandom Color = "random"
)

// AuthTest can be set to true to test an auth token.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/auth#auth_test
var AuthTest = false

// AuthTestResponse will contain the server response of any
// API calls if AuthTest=true.
var AuthTestResponse = map[string]interface{}{}

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
	b.Write([]byte(body))

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
// Do can be used to perform the request created with NewRequest, as the latter
// it should be used only for API requests not implemented in this library.
func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
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
				io.Copy(w, resp.Body)
			} else {
				err = json.NewDecoder(resp.Body).Decode(v)
			}
		}
	}
	return resp, err
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
