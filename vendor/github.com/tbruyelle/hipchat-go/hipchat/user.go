package hipchat

import (
	"fmt"
	"net/http"
)

// MessageRequest represents a HipChat private message to user.
type MessageRequest struct {
	Message       string `json:"message,omitempty"`
	Notify        bool   `json:"notify,omitempty"`
	MessageFormat string `json:"message_format,omitempty"`
}

// UserPresence represents the HipChat user's presence.
type UserPresence struct {
	Status   string `json:"status"`
	Idle     int    `json:"idle"`
	Show     string `json:"show"`
	IsOnline bool   `json:"is_online"`
}

const (
	// UserPresenceShowAway show status away
	UserPresenceShowAway = "away"

	// UserPresenceShowChat show status available to chat
	UserPresenceShowChat = "chat"

	// UserPresenceShowDnd show status do not disturb
	UserPresenceShowDnd = "dnd"

	// UserPresenceShowXa show status xa?
	UserPresenceShowXa = "xa"
)

// UpdateUserRequest represents a HipChat user update request body.
type UpdateUserRequest struct {
	Name        string                    `json:"name"`
	Presence    UpdateUserPresenceRequest `json:"presence"`
	MentionName string                    `json:"mention_name"`
	Email       string                    `json:"email"`
}

// UpdateUserPresenceRequest represents the HipChat user's presence update request body.
type UpdateUserPresenceRequest struct {
	Status string `json:"status"`
	Show   string `json:"show"`
}

// User represents the HipChat user.
type User struct {
	XmppJid      string       `json:"xmpp_jid"`
	IsDeleted    bool         `json:"is_deleted"`
	Name         string       `json:"name"`
	LastActive   string       `json:"last_active"`
	Title        string       `json:"title"`
	Presence     UserPresence `json:"presence"`
	Created      string       `json:"created"`
	ID           int          `json:"id"`
	MentionName  string       `json:"mention_name"`
	IsGroupAdmin bool         `json:"is_group_admin"`
	Timezone     string       `json:"timezone"`
	IsGuest      bool         `json:"is_guest"`
	Email        string       `json:"email"`
	PhotoURL     string       `json:"photo_url"`
	Links        Links        `json:"links"`
}

// Users represents the API return of a collection of Users plus metadata
type Users struct {
	Items      []User `json:"items"`
	StartIndex int    `json:"start_index"`
	MaxResults int    `json:"max_results"`
	Links      Links  `json:"links"`
}

// UserService gives access to the user related methods of the API.
type UserService struct {
	client *Client
}

// ShareFile sends a file to the user specified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/share_file_with_user
func (u *UserService) ShareFile(id string, shareFileReq *ShareFileRequest) (*http.Response, error) {
	req, err := u.client.NewFileUploadRequest("POST", fmt.Sprintf("user/%s/share/file", id), shareFileReq)
	if err != nil {
		return nil, err
	}

	return u.client.Do(req, nil)
}

// View fetches a user's details.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/view_user
func (u *UserService) View(id string) (*User, *http.Response, error) {
	req, err := u.client.NewRequest("GET", fmt.Sprintf("user/%s", id), nil, nil)

	userDetails := new(User)
	resp, err := u.client.Do(req, &userDetails)
	if err != nil {
		return nil, resp, err
	}
	return userDetails, resp, nil
}

// Message sends a private message to the user specified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/private_message_user
func (u *UserService) Message(id string, msgReq *MessageRequest) (*http.Response, error) {
	req, err := u.client.NewRequest("POST", fmt.Sprintf("user/%s/message", id), nil, msgReq)
	if err != nil {
		return nil, err
	}

	return u.client.Do(req, nil)
}

// UserListOptions specified the parameters to the UserService.List method.
type UserListOptions struct {
	ListOptions
	// Include active guest users in response.
	IncludeGuests bool `url:"include-guests,omitempty"`
	// Include deleted users in response.
	IncludeDeleted bool `url:"include-deleted,omitempty"`
}

// List returns all users in the group.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/get_all_users
func (u *UserService) List(opt *UserListOptions) ([]User, *http.Response, error) {
	req, err := u.client.NewRequest("GET", "user", opt, nil)

	users := new(Users)
	resp, err := u.client.Do(req, &users)
	if err != nil {
		return nil, resp, err
	}
	return users.Items, resp, nil
}

// Update a user
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/update_user
func (u *UserService) Update(id string, user *UpdateUserRequest) (*http.Response, error) {
	req, err := u.client.NewRequest("PUT", fmt.Sprintf("user/%s", id), nil, user)
	if err != nil {
		return nil, err
	}

	return u.client.Do(req, nil)
}
