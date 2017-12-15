package hipchat

import (
	"net/http"
)

// EmoticonService gives access to the emoticon related part of the API.
type EmoticonService struct {
	client *Client
}

// Emoticons represents a list of hipchat emoticons.
type Emoticons struct {
	Items      []Emoticon `json:"items"`
	StartIndex int        `json:"startIndex"`
	MaxResults int        `json:"maxResults"`
	Links      PageLinks  `json:"links"`
}

// Emoticon represents a hipchat emoticon.
type Emoticon struct {
	ID       int    `json:"id"`
	URL      string `json:"url"`
	Links    Links  `json:"links"`
	Shortcut string `json:"shortcut"`
}

// EmoticonsListOptions specifies the optionnal parameters of the EmoticonService.List
// method.
type EmoticonsListOptions struct {
	ListOptions

	// The type of emoticons to get (global, group or all)
	Type string `url:"type,omitempty"`
}

// List returns the list of all the emoticons
//
// HipChat api docs : https://www.hipchat.com/docs/apiv2/method/get_all_emoticons
func (e *EmoticonService) List(opt *EmoticonsListOptions) (*Emoticons, *http.Response, error) {
	req, err := e.client.NewRequest("GET", "emoticon", opt, nil)
	if err != nil {
		return nil, nil, err
	}

	emoticons := new(Emoticons)
	resp, err := e.client.Do(req, emoticons)
	if err != nil {
		return nil, resp, err
	}
	return emoticons, resp, nil
}
