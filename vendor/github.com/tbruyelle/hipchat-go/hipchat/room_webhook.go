// handling Webhook data

package hipchat

import (
	"fmt"
	"net/http"
)

// Response Types

// Webhook represents a HipChat webhook.
type Webhook struct {
	Links   Links  `json:"links"`
	Name    string `json:"name"`
	Key     string `json:"key,omitempty"`
	Event   string `json:"event"`
	Pattern string `json:"pattern"`
	URL     string `json:"url"`
	ID      int    `json:"id,omitempty"`
}

// WebhookList represents a HipChat webhook list.
type WebhookList struct {
	Webhooks   []Webhook `json:"items"`
	StartIndex int       `json:"startIndex"`
	MaxResults int       `json:"maxResults"`
	Links      PageLinks `json:"links"`
}

// Request Types

// ListWebhooksOptions represents options for ListWebhooks method.
type ListWebhooksOptions struct {
	ListOptions
}

// CreateWebhookRequest represents the body of the CreateWebhook method.
type CreateWebhookRequest struct {
	Name    string `json:"name"`
	Key     string `json:"key,omitempty"`
	Event   string `json:"event"`
	Pattern string `json:"pattern"`
	URL     string `json:"url"`
}

// ListWebhooks returns all the webhooks for a given room.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/get_all_webhooks
func (r *RoomService) ListWebhooks(id interface{}, opt *ListWebhooksOptions) (*WebhookList, *http.Response, error) {
	u := fmt.Sprintf("room/%v/webhook", id)
	req, err := r.client.NewRequest("GET", u, opt, nil)
	if err != nil {
		return nil, nil, err
	}
	whList := new(WebhookList)

	resp, err := r.client.Do(req, whList)
	if err != nil {
		return nil, resp, err
	}
	return whList, resp, nil
}

// DeleteWebhook removes the given webhook.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/delete_webhook
func (r *RoomService) DeleteWebhook(id interface{}, webhookID interface{}) (*http.Response, error) {
	req, err := r.client.NewRequest("DELETE", fmt.Sprintf("room/%v/webhook/%v", id, webhookID), nil, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// CreateWebhook creates a new webhook.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/create_webhook
func (r *RoomService) CreateWebhook(id interface{}, roomReq *CreateWebhookRequest) (*Webhook, *http.Response, error) {
	req, err := r.client.NewRequest("POST", fmt.Sprintf("room/%v/webhook", id), nil, roomReq)
	if err != nil {
		return nil, nil, err
	}

	wh := new(Webhook)

	resp, err := r.client.Do(req, wh)
	if err != nil {
		return nil, resp, err
	}

	return wh, resp, nil
}
