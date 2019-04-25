package hipchat

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// ClientCredentials represents the OAuth2 client ID and secret for an integration
type ClientCredentials struct {
	ClientID     string
	ClientSecret string
}

// OAuthAccessToken represents a newly created Hipchat OAuth access token
type OAuthAccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   uint32 `json:"expires_in"`
	GroupID     uint32 `json:"group_id"`
	GroupName   string `json:"group_name"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

// CreateClient creates a new client from this OAuth token
func (t *OAuthAccessToken) CreateClient() *Client {
	return NewClient(t.AccessToken)
}

// GenerateToken returns back an access token for a given integration's client ID and client secret
//
//  HipChat API documentation: https://www.hipchat.com/docs/apiv2/method/generate_token
func (c *Client) GenerateToken(credentials ClientCredentials, scopes []string) (*OAuthAccessToken, *http.Response, error) {
	rel, err := url.Parse("oauth/token")

	if err != nil {
		return nil, nil, err
	}

	u := c.BaseURL.ResolveReference(rel)

	params := url.Values{"grant_type": {"client_credentials"},
		"scope": {strings.Join(scopes, " ")}}
	req, err := http.NewRequest("POST", u.String(), strings.NewReader(params.Encode()))

	if err != nil {
		return nil, nil, err
	}

	req.SetBasicAuth(credentials.ClientID, credentials.ClientSecret)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)

	if err != nil {
		return nil, resp, err
	}

	if resp.StatusCode != 200 {
		content, readerr := ioutil.ReadAll(resp.Body)

		if readerr != nil {
			content = []byte("Unknown error")
		}

		return nil, resp, fmt.Errorf("Couldn't retrieve access token: %s", content)
	}

	content, err := ioutil.ReadAll(resp.Body)

	var token OAuthAccessToken
	json.Unmarshal(content, &token)

	return &token, resp, nil
}

const (
	// ScopeAdminGroup - Perform group administrative tasks
	ScopeAdminGroup = "admin_group"

	// ScopeAdminRoom - Perform room administrative tasks
	ScopeAdminRoom = "admin_room"

	// ScopeImportData - Import users, rooms, and chat history. Only available for select add-ons.
	ScopeImportData = "import_data"

	// ScopeManageRooms - Create, update, and remove rooms
	ScopeManageRooms = "manage_rooms"

	// ScopeSendMessage - Send private one-on-one messages
	ScopeSendMessage = "send_message"

	// ScopeSendNotification - Send room notifications
	ScopeSendNotification = "send_notification"

	// ScopeViewGroup - View users, rooms, and other group information
	ScopeViewGroup = "view_group"

	// ScopeViewMessages - View messages from chat rooms and private chats you have access to
	ScopeViewMessages = "view_messages"

	// ScopeViewRoom - View room information and participants, but not history
	ScopeViewRoom = "view_room"
)
