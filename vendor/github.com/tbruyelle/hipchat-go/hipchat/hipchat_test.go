package hipchat

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

var (
	mux    *http.ServeMux
	server *httptest.Server
	client *Client
)

// setup sets up a test HTTP server and a hipchat.Client configured to talk
// to that test server.
// Tests should register handlers on mux which provide mock responses for
// the API method being tested.
func setup() {
	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	// github client configured to use test server
	client = NewClient("AuthToken")
	url, _ := url.Parse(server.URL)
	client.BaseURL = url
}

// teardown closes the test HTTP server.
func teardown() {
	server.Close()
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

type values map[string]string

func testFormValues(t *testing.T, r *http.Request, values values) {
	want := url.Values{}
	for k, v := range values {
		want.Add(k, v)
	}

	r.ParseForm()
	if got := r.Form; !reflect.DeepEqual(got, want) {
		t.Errorf("Request parameters: %v, want %v", got, want)
	}
}

func testHeader(t *testing.T, r *http.Request, header string, want string) {
	if got := r.Header.Get(header); got != want {
		t.Errorf("Header.Get(%q) returned %s, want %s", header, got, want)
	}
}

func TestNewClient(t *testing.T) {
	authToken := "AuthToken"

	c := NewClient(authToken)

	if c.authToken != authToken {
		t.Errorf("NewClient authToken %s, want %s", c.authToken, authToken)
	}
	if c.BaseURL.String() != defaultBaseURL {
		t.Errorf("NewClient BaseURL %s, want %s", c.BaseURL.String(), defaultBaseURL)
	}
	if c.client != http.DefaultClient {
		t.Errorf("SetHTTPClient client %v, want %p", c.client, http.DefaultClient)
	}
}

func TestSetHTTPClient(t *testing.T) {
	c := NewClient("AuthToken")

	httpClient := new(http.Client)
	c.SetHTTPClient(httpClient)

	if c.client != httpClient {
		t.Errorf("SetHTTPClient client %v, want %p", c.client, httpClient)
	}
}

type customHTTPClient struct{}

func (c customHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, nil
}

func TestSetCustomHTTPClient(t *testing.T) {
	c := NewClient("AuthToken")

	httpClient := new(customHTTPClient)
	c.SetHTTPClient(httpClient)

	if c.client != httpClient {
		t.Errorf("SetHTTPClient client %v, want %p", c.client, httpClient)
	}
}

func TestSetHTTPClient_NilHTTPClient(t *testing.T) {
	c := NewClient("AuthToken")

	c.SetHTTPClient(nil)

	if c.client != http.DefaultClient {
		t.Errorf("SetHTTPClient client %v, want %p", c.client, http.DefaultClient)
	}
}

func TestNewRequest(t *testing.T) {
	c := NewClient("AuthToken")

	inURL, outURL := "foo", defaultBaseURL+"foo?max-results=100&start-index=1"
	opt := &ListOptions{StartIndex: 1, MaxResults: 100}
	inBody, outBody := &NotificationRequest{Message: "Hello"}, `{"message":"Hello"}`+"\n"
	r, _ := c.NewRequest("GET", inURL, opt, inBody)

	if r.URL.String() != outURL {
		t.Errorf("NewRequest URL %s, want %s", r.URL.String(), outURL)
	}
	body, _ := ioutil.ReadAll(r.Body)
	if string(body) != outBody {
		t.Errorf("NewRequest body %s, want %s", body, outBody)
	}
	authorization := r.Header.Get("Authorization")
	if authorization != "Bearer "+c.authToken {
		t.Errorf("NewRequest authorization header %s, want %s", authorization, "Bearer "+c.authToken)
	}
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("NewRequest Content-Type header %s, want application/json", contentType)
	}
}

func TestNewRequest_AuthTestEnabled(t *testing.T) {
	AuthTest = true
	defer func() { AuthTest = false }()
	c := NewClient("AuthToken")

	inURL, outURL := "foo", defaultBaseURL+"foo?auth_test=true"
	r, _ := c.NewRequest("GET", inURL, nil, nil)

	if r.URL.String() != outURL {
		t.Errorf("NewRequest URL %s, want %s", r.URL.String(), outURL)
	}
}

func TestDo(t *testing.T) {
	setup()
	defer teardown()

	type foo struct {
		Bar int
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if m := "GET"; m != r.Method {
			t.Errorf("Request method = %v, want %v", r.Method, m)
		}
		fmt.Fprintf(w, `{"Bar":1}`)
	})
	req, _ := client.NewRequest("GET", "/", nil, nil)
	body := new(foo)

	_, err := client.Do(req, body)

	if err != nil {
		t.Fatal(err)
	}
	want := &foo{Bar: 1}
	if !reflect.DeepEqual(body, want) {
		t.Errorf("Response body = %v, want %v", body, want)
	}
}

func TestDo_AuthTestEnabled(t *testing.T) {
	AuthTest = true
	defer func() { AuthTest = false }()
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if m := "GET"; m != r.Method {
			t.Errorf("Request method = %v, want %v", r.Method, m)
		}
		if r.URL.Query().Get("auth_test") == "true" {
			fmt.Fprintf(w, `{"success":{ "code": 202, "type": "Accepted", "message": "This auth_token has access to use this method." }}`)
		} else {
			fmt.Fprintf(w, `{"Bar":1}`)
		}
	})
	req, _ := client.NewRequest("GET", "/", nil, nil)

	_, err := client.Do(req, nil)

	if err != nil {
		t.Fatal(err)
	}
	if _, ok := AuthTestResponse["success"]; !ok {
		t.Errorf("Response body = %v, want succeed", AuthTestResponse)
	}
}
