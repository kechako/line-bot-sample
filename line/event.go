package line

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

var (
	LineChannelID     = os.Getenv("LINE_CHANNEL_ID")
	LineChannelSecret = os.Getenv("LINE_CHANNEL_SECRET")
	MID               = os.Getenv("MID")
)

const (
	EventPostUrl = "https://trialbot-api.line.me/v1/events"
)

type Event struct {
	To        []string    `json:"to"`
	ToChannel int         `json:"toChannel"`
	EventType string      `json:"eventType"`
	Content   interface{} `json:"content"`
}

type Text struct {
	ContentType int    `json:"contentType"`
	ToType      int    `json:"toType"`
	Text        string `json:"text"`
}

func NewEvent(to []string) *Event {
	return &Event{
		To:        to,
		ToChannel: ToChannel,
		EventType: EventTypeSendMessage,
	}
}

func NewText(text string) *Text {
	return &Text{
		ContentType: ContentTypeText,
		ToType:      1,
		Text:        text,
	}
}

func (e *Event) Send(client http.Client) (*http.Response, error) {
	body, err := json.Marshal(e)
	if err != nil {
		return nil, errors.Wrap(err, "Json marshaling error")
	}

	req, err := http.NewRequest("POST", EventPostUrl, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "Can not create a new request")
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Line-ChannelID", LineChannelID)
	req.Header.Set("X-Line-ChannelSecret", LineChannelSecret)
	req.Header.Set("X-Line-Trusted-User-With-ACL", MID)

	return client.Do(req)
}
