package hipchat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// RoomService gives access to the room related methods of the API.
type RoomService struct {
	client *Client
}

// Rooms represents a HipChat room list.
type Rooms struct {
	Items      []Room    `json:"items"`
	StartIndex int       `json:"startIndex"`
	MaxResults int       `json:"maxResults"`
	Links      PageLinks `json:"links"`
}

// Room represents a HipChat room.
type Room struct {
	ID                int            `json:"id"`
	Links             RoomLinks      `json:"links"`
	Name              string         `json:"name"`
	XMPPJid           string         `json:"xmpp_jid"`
	Statistics        RoomStatistics `json:"statistics"`
	Created           string         `json:"created"`
	IsArchived        bool           `json:"is_archived"`
	Privacy           string         `json:"privacy"`
	IsGuestAccessible bool           `json:"is_guess_accessible"`
	Topic             string         `json:"topic"`
	Participants      []User         `json:"participants"`
	Owner             User           `json:"owner"`
	GuestAccessURL    string         `json:"guest_access_url"`
}

// RoomStatistics represents the HipChat room statistics.
type RoomStatistics struct {
	Links        Links  `json:"links"`
	MessagesSent int    `json:"messages_sent,omitempty"`
	LastActive   string `json:"last_active,omitempty"`
}

// CreateRoomRequest represents a HipChat room creation request.
type CreateRoomRequest struct {
	Topic       string `json:"topic,omitempty"`
	GuestAccess bool   `json:"guest_access,omitempty"`
	Name        string `json:"name,omitempty"`
	OwnerUserID string `json:"owner_user_id,omitempty"`
	Privacy     string `json:"privacy,omitempty"`
}

// UpdateRoomRequest represents a HipChat room update request.
type UpdateRoomRequest struct {
	Name          string `json:"name"`
	Topic         string `json:"topic"`
	IsGuestAccess bool   `json:"is_guest_accessible"`
	IsArchived    bool   `json:"is_archived"`
	Privacy       string `json:"privacy"`
	Owner         ID     `json:"owner"`
}

// RoomLinks represents the HipChat room links.
type RoomLinks struct {
	Links
	Webhooks     string `json:"webhooks"`
	Members      string `json:"members"`
	Participants string `json:"participants"`
}

// NotificationRequest represents a HipChat room notification request.
type NotificationRequest struct {
	Color         Color  `json:"color,omitempty"`
	Message       string `json:"message,omitempty"`
	Notify        bool   `json:"notify,omitempty"`
	MessageFormat string `json:"message_format,omitempty"`
	From          string `json:"from,omitempty"`
	Card          *Card  `json:"card,omitempty"`
}

// RoomMessageRequest represents a Hipchat room message request.
type RoomMessageRequest struct {
	Message string `json:"message"`
}

// Card is used to send information as messages to Hipchat rooms
type Card struct {
	Style       string          `json:"style"`
	Description CardDescription `json:"description"`
	Format      string          `json:"format,omitempty"`
	URL         string          `json:"url,omitempty"`
	Title       string          `json:"title"`
	Thumbnail   *Thumbnail      `json:"thumbnail,omitempty"`
	Activity    *Activity       `json:"activity,omitempty"`
	Attributes  []Attribute     `json:"attributes,omitempty"`
	ID          string          `json:"id,omitempty"`
	Icon        *Icon           `json:"icon,omitempty"`
}

const (
	// CardStyleFile represents a Card notification related to a file
	CardStyleFile = "file"

	// CardStyleImage represents a Card notification related to an image
	CardStyleImage = "image"

	// CardStyleApplication represents a Card notification related to an application
	CardStyleApplication = "application"

	// CardStyleLink represents a Card notification related to a link
	CardStyleLink = "link"

	// CardStyleMedia represents a Card notiifcation related to media
	CardStyleMedia = "media"
)

// CardDescription represents the main content of the Card
type CardDescription struct {
	Format string
	Value  string
}

// MarshalJSON serializes a CardDescription into JSON
func (c CardDescription) MarshalJSON() ([]byte, error) {
	if c.Format == "" {
		return json.Marshal(c.Value)
	}

	obj := make(map[string]string)
	obj["format"] = c.Format
	obj["value"] = c.Value

	return json.Marshal(obj)
}

// UnmarshalJSON deserializes a JSON-serialized CardDescription
func (c *CardDescription) UnmarshalJSON(data []byte) error {
	// Compact the JSON to make it easier to process below
	buffer := bytes.NewBuffer([]byte{})
	err := json.Compact(buffer, data)
	if err != nil {
		return err
	}
	data = buffer.Bytes()

	// Since Description can be either a string value or an object, we
	// must check and deserialize appropriately

	if data[0] == 123 { // == }
		obj := make(map[string]string)

		err = json.Unmarshal(data, &obj)
		if err != nil {
			return err
		}

		c.Format = obj["format"]
		c.Value = obj["value"]
	} else {
		c.Format = ""
		err = json.Unmarshal(data, &c.Value)
	}

	if err != nil {
		return err
	}

	return nil
}

// Icon represents an icon
type Icon struct {
	URL   string `json:"url"`
	URL2x string `json:"url@2x,omitempty"`
}

// Thumbnail represents a thumbnail image
type Thumbnail struct {
	URL    string `json:"url"`
	URL2x  string `json:"url@2x,omitempty"`
	Width  uint   `json:"width,omitempty"`
	Height uint   `json:"height,omitempty"`
}

// Attribute represents an attribute on a Card
type Attribute struct {
	Label string         `json:"label,omitempty"`
	Value AttributeValue `json:"value"`
}

// AttributeValue represents the value of an attribute
type AttributeValue struct {
	URL   string `json:"url,omitempty"`
	Style string `json:"style,omitempty"`
	Type  string `json:"type,omitempty"`
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
	Icon  *Icon  `json:"icon,omitempty"`
}

// Activity represents an activity that occurred
type Activity struct {
	Icon *Icon  `json:"icon,omitempty"`
	HTML string `json:"html,omitempty"`
}

// ShareFileRequest represents a HipChat room file share request.
type ShareFileRequest struct {
	Path     string `json:"path"`
	Filename string `json:"filename,omitempty"`
	Message  string `json:"message,omitempty"`
}

// History represents a HipChat room chat history.
type History struct {
	Items      []Message `json:"items"`
	StartIndex int       `json:"startIndex"`
	MaxResults int       `json:"maxResults"`
	Links      PageLinks `json:"links"`
}

// Message represents a HipChat message.
type Message struct {
	Date          string      `json:"date"`
	From          interface{} `json:"from"` // string | obj <- weak
	ID            string      `json:"id"`
	Mentions      []User      `json:"mentions"`
	Message       string      `json:"message"`
	MessageFormat string      `json:"message_format"`
	Type          string      `json:"type"`
}

// SetTopicRequest represents a hipchat update topic request
type SetTopicRequest struct {
	Topic string `json:"topic"`
}

// InviteRequest represents a hipchat invite to room request
type InviteRequest struct {
	Reason string `json:"reason"`
}

// AddMemberRequest represents a HipChat add member request
type AddMemberRequest struct {
	Roles []string `json:"roles,omitempty"`
}

// GlanceRequest represents a HipChat room ui glance
type GlanceRequest struct {
	Key        string             `json:"key"`
	Name       GlanceName         `json:"name"`
	Target     string             `json:"target"`
	QueryURL   string             `json:"queryUrl,omitempty"`
	Icon       Icon               `json:"icon"`
	Conditions []*GlanceCondition `json:"conditions,omitempty"`
}

// GlanceName represents a glance name
type GlanceName struct {
	Value string `json:"value"`
	I18n  string `json:"i18n,omitempty"`
}

// GlanceCondition represents a condition to determine whether a glance is displayed
type GlanceCondition struct {
	Condition string            `json:"condition"`
	Params    map[string]string `json:"params"`
	Invert    bool              `json:"invert"`
}

// GlanceUpdateRequest represents a HipChat room ui glance update request
type GlanceUpdateRequest struct {
	Glance []*GlanceUpdate `json:"glance"`
}

// GlanceUpdate represents a component of a HipChat room ui glance update
type GlanceUpdate struct {
	Key     string        `json:"key"`
	Content GlanceContent `json:"content"`
}

// GlanceContent is a component of a Glance
type GlanceContent struct {
	Status   *GlanceStatus  `json:"status,omitempty"`
	Metadata interface{}    `json:"metadata,omitempty"`
	Label    AttributeValue `json:"label"` // AttributeValue{Type, Label}
}

// GlanceStatus is a status field component of a GlanceContent
type GlanceStatus struct {
	Type  string      `json:"type"`  // "lozenge" | "icon"
	Value interface{} `json:"value"` // AttributeValue{Type, Label} | Icon{URL, URL2x}
}

// UnmarshalJSON deserializes a JSON-serialized GlanceStatus
func (gs *GlanceStatus) UnmarshalJSON(data []byte) error {
	// Compact the JSON to make it easier to process below
	buffer := bytes.NewBuffer([]byte{})
	err := json.Compact(buffer, data)
	if err != nil {
		return err
	}
	data = buffer.Bytes()

	// Since Value can be either an AttributeValue or an Icon, we
	// must check and deserialize appropriately
	obj := make(map[string]interface{})

	err = json.Unmarshal(data, &obj)
	if err != nil {
		return err
	}

	for _, field := range []string{"type", "value"} {
		if obj[field] == nil {
			return fmt.Errorf("missing %s field", field)
		}
	}

	gs.Type = obj["type"].(string)
	val := obj["value"].(map[string]interface{})

	valueMap := map[string][]string{
		"lozenge": []string{"type", "label"},
		"icon":    []string{"url", "url@2x"},
	}

	if valueMap[gs.Type] == nil {
		return fmt.Errorf("invalid GlanceStatus type: %s", gs.Type)
	}

	for _, field := range valueMap[gs.Type] {
		if val[field] == nil {
			return fmt.Errorf("%s missing %s field", gs.Type, field)
		}
		_, ok := val[field].(string)
		if !ok {
			return fmt.Errorf("could not convert %s field %s to string", gs.Type, field)
		}
	}

	// Can safely perform type coercion
	switch gs.Type {
	case "lozenge":
		gs.Value = AttributeValue{Type: val["type"].(string), Label: val["label"].(string)}
	case "icon":
		gs.Value = Icon{URL: val["url"].(string), URL2x: val["url@2x"].(string)}
	}

	return nil
}

// AddAttribute adds an attribute to a Card
func (c *Card) AddAttribute(mainLabel, subLabel, url, iconURL string) {
	attr := Attribute{Label: mainLabel}
	attr.Value = AttributeValue{Label: subLabel, URL: url, Icon: &Icon{URL: iconURL}}

	c.Attributes = append(c.Attributes, attr)
}

// RoomsListOptions specifies the optional parameters of the RoomService.List
// method.
type RoomsListOptions struct {
	ListOptions
	ExpandOptions

	// Include private rooms in the result, API defaults to true
	IncludePrivate bool `url:"include-private,omitempty"`

	// Include archived rooms in the result, API defaults to false
	IncludeArchived bool `url:"include-archived,omitempty"`
}

// List returns all the rooms authorized.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/get_all_rooms
func (r *RoomService) List(opt *RoomsListOptions) (*Rooms, *http.Response, error) {
	req, err := r.client.NewRequest("GET", "room", opt, nil)
	if err != nil {
		return nil, nil, err
	}

	rooms := new(Rooms)
	resp, err := r.client.Do(req, rooms)
	if err != nil {
		return nil, resp, err
	}
	return rooms, resp, nil
}

// Get returns the room specified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/get_room
func (r *RoomService) Get(id string) (*Room, *http.Response, error) {
	req, err := r.client.NewRequest("GET", fmt.Sprintf("room/%s", id), nil, nil)
	if err != nil {
		return nil, nil, err
	}

	room := new(Room)
	resp, err := r.client.Do(req, room)
	if err != nil {
		return nil, resp, err
	}
	return room, resp, nil
}

// GetStatistics returns the room statistics pecified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/get_room_statistics
func (r *RoomService) GetStatistics(id string) (*RoomStatistics, *http.Response, error) {
	req, err := r.client.NewRequest("GET", fmt.Sprintf("room/%s/statistics", id), nil, nil)
	if err != nil {
		return nil, nil, err
	}

	roomStatistics := new(RoomStatistics)
	resp, err := r.client.Do(req, roomStatistics)
	if err != nil {
		return nil, resp, err
	}
	return roomStatistics, resp, nil
}

// Notification sends a notification to the room specified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/send_room_notification
func (r *RoomService) Notification(id string, notifReq *NotificationRequest) (*http.Response, error) {
	req, err := r.client.NewRequest("POST", fmt.Sprintf("room/%s/notification", id), nil, notifReq)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// Message sends a message to the room specified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/send_message
func (r *RoomService) Message(id string, msgReq *RoomMessageRequest) (*http.Response, error) {
	req, err := r.client.NewRequest("POST", fmt.Sprintf("room/%s/message", id), nil, msgReq)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// ShareFile sends a file to the room specified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/share_file_with_room
func (r *RoomService) ShareFile(id string, shareFileReq *ShareFileRequest) (*http.Response, error) {
	req, err := r.client.NewFileUploadRequest("POST", fmt.Sprintf("room/%s/share/file", id), shareFileReq)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// Create creates a new room.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/create_room
func (r *RoomService) Create(roomReq *CreateRoomRequest) (*Room, *http.Response, error) {
	req, err := r.client.NewRequest("POST", "room", nil, roomReq)
	if err != nil {
		return nil, nil, err
	}

	room := new(Room)
	resp, err := r.client.Do(req, room)
	if err != nil {
		return nil, resp, err
	}
	return room, resp, nil
}

// Delete deletes an existing room.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/delete_room
func (r *RoomService) Delete(id string) (*http.Response, error) {
	req, err := r.client.NewRequest("DELETE", fmt.Sprintf("room/%s", id), nil, nil)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// Update updates an existing room.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/update_room
func (r *RoomService) Update(id string, roomReq *UpdateRoomRequest) (*http.Response, error) {
	req, err := r.client.NewRequest("PUT", fmt.Sprintf("room/%s", id), nil, roomReq)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// HistoryOptions represents a HipChat room chat history request.
type HistoryOptions struct {
	ListOptions
	ExpandOptions

	// Either the latest date to fetch history for in ISO-8601 format, or 'recent' to fetch
	// the latest 75 messages. Paging isn't supported for 'recent', however they are real-time
	// values, whereas date queries may not include the most recent messages.
	Date string `url:"date,omitempty"`

	// Your timezone. Must be a supported timezone
	Timezone string `url:"timezone,omitempty"`

	// Reverse the output such that the oldest message is first.
	// For consistent paging, set to 'false'.
	Reverse bool `url:"reverse,omitempty"`

	// Either the earliest date to fetch history for the ISO-8601 format string,
	// or leave blank to disable this filter.
	// to be effective, the API call requires Date also be filled in with an ISO-8601 format string.
	EndDate string `url:"end-date,omitempty"`

	// Include records about deleted messages into results (body of a message isn't returned).  Set to 'true'.
	IncludeDeleted bool `url:"include_deleted,omitempty"`
}

// History fetches a room's chat history.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/view_room_history
func (r *RoomService) History(id string, opt *HistoryOptions) (*History, *http.Response, error) {
	u := fmt.Sprintf("room/%s/history", id)
	req, err := r.client.NewRequest("GET", u, opt, nil)
	h := new(History)
	resp, err := r.client.Do(req, &h)
	if err != nil {
		return nil, resp, err
	}
	return h, resp, nil
}

// LatestHistoryOptions represents a HipChat room chat latest history request.
type LatestHistoryOptions struct {

	// The maximum number of messages to return.
	MaxResults int `url:"max-results,omitempty"`

	// Your timezone. Must be a supported timezone.
	Timezone string `url:"timezone,omitempty"`

	// The id of the message that is oldest in the set of messages to be returned.
	// The server will not return any messages that chronologically precede this message.
	NotBefore string `url:"not-before,omitempty"`
}

// Latest fetches a room's chat history.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/view_recent_room_history
func (r *RoomService) Latest(id string, opt *LatestHistoryOptions) (*History, *http.Response, error) {
	u := fmt.Sprintf("room/%s/history/latest", id)
	req, err := r.client.NewRequest("GET", u, opt, nil)
	h := new(History)
	resp, err := r.client.Do(req, &h)
	if err != nil {
		return nil, resp, err
	}
	return h, resp, nil
}

// SetTopic sets Room topic.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/set_topic
func (r *RoomService) SetTopic(id string, topic string) (*http.Response, error) {
	topicReq := &SetTopicRequest{Topic: topic}

	req, err := r.client.NewRequest("PUT", fmt.Sprintf("room/%s/topic", id), nil, topicReq)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// Invite someone to the Room.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/invite_user
func (r *RoomService) Invite(room string, user string, reason string) (*http.Response, error) {
	reasonReq := &InviteRequest{Reason: reason}

	req, err := r.client.NewRequest("POST", fmt.Sprintf("room/%s/invite/%s", room, user), nil, reasonReq)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// CreateGlance creates a glance in the room specified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/create_room_glance
func (r *RoomService) CreateGlance(id string, glanceReq *GlanceRequest) (*http.Response, error) {
	req, err := r.client.NewRequest("PUT", fmt.Sprintf("room/%s/extension/glance/%s", id, glanceReq.Key), nil, glanceReq)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// DeleteGlance deletes a glance in the room specified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/delete_room_glance
func (r *RoomService) DeleteGlance(id string, glanceReq *GlanceRequest) (*http.Response, error) {
	req, err := r.client.NewRequest("DELETE", fmt.Sprintf("room/%s/extension/glance/%s", id, glanceReq.Key), nil, nil)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// UpdateGlance sends a glance update to the room specified by the id.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/room_addon_ui_update
func (r *RoomService) UpdateGlance(id string, glanceUpdateReq *GlanceUpdateRequest) (*http.Response, error) {
	req, err := r.client.NewRequest("POST", fmt.Sprintf("addon/ui/room/%s", id), nil, glanceUpdateReq)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// AddMember adds a member to a private room and sends member's unavailable presence to all room members asynchronously.
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/add_member
func (r *RoomService) AddMember(roomID string, userID string, addMemberReq *AddMemberRequest) (*http.Response, error) {
	req, err := r.client.NewRequest("PUT", fmt.Sprintf("room/%s/member/%s", roomID, userID), nil, addMemberReq)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}

// RemoveMember removes a member from a private room
//
// HipChat API docs: https://www.hipchat.com/docs/apiv2/method/remove_member
func (r *RoomService) RemoveMember(roomID string, userID string) (*http.Response, error) {
	req, err := r.client.NewRequest("DELETE", fmt.Sprintf("room/%s/member/%s", roomID, userID), nil, nil)
	if err != nil {
		return nil, err
	}

	return r.client.Do(req, nil)
}
