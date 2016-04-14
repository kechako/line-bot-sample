package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

var (
	LineChannelID     = os.Getenv("LINE_CHANNEL_ID")
	LineChannelSecret = os.Getenv("LINE_CHANNEL_SECRET")
	MID               = os.Getenv("MID")
)

const (
	EventTypeMessage     = "138311609000106303"
	EventTypeOperation   = "138311609100106403"
	EventTypeSendMessage = "138311608800106203"
)

const (
	ContentTypeText = 1 + iota
	ContentTypeImage
	ContentTypeVideo
	ContentTypeAudio
	_
	_
	ContentTypeLocation
	ContentTypeSticker
	ContentTypeContact
)

type Result struct {
	From        string          `json:"from"`
	FromChannel int             `json:"fromChannel"`
	To          []string        `json:"to"`
	ToChannel   int             `json:"toChannel"`
	EventType   string          `json:"eventType"`
	Id          string          `json:"id"`
	CreatedTime int             `json:"createdTime"`
	Content     json.RawMessage `json:"content"`
}

type Request struct {
	Result []Result `json:result`
}

type Location struct {
	Title     string  `json:"title"`
	Address   string  `json:"address"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
}

type Message struct {
	Id              string          `json:"id"`
	ContentType     int             `json:"contentType"`
	From            string          `json:"from"`
	CreatedTime     int             `json:"createdTime"`
	To              []string        `json:"to"`
	ToType          int             `json:"toType"`
	ContentMetadata json.RawMessage `json:"contentMetadata"`
	Text            string          `json:"text"`
	Location        Location        `json:"location"`
}

type Operation struct {
	Revision int      `json:"revision"`
	OpType   int      `json:"opType"`
	params   []string `json:"params"`
}

const (
	ToChannel = 1383378250
)

type Event struct {
	To        []string    `json:"to"`
	ToChannel int         `json:"toChannel"`
	EventType string      `json:"eventType"`
	Content   interface{} `json:"content"`
}

func NewEvent(to []string) *Event {
	return &Event{
		To:        to,
		ToChannel: ToChannel,
		EventType: EventTypeSendMessage,
	}
}

func (e *Event) Send(client *http.Client) (*http.Response, error) {
	body, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("Message sending error : %v\n", err)
	}

	req, err := http.NewRequest("POST", "https://trialbot-api.line.me/v1/events", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Can't create new request : %v\n", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Line-ChannelID", LineChannelID)
	req.Header.Set("X-Line-ChannelSecret", LineChannelSecret)
	req.Header.Set("X-Line-Trusted-User-With-ACL", MID)

	return client.Do(req)
}

type Text struct {
	ContentType int    `json:"contentType"`
	ToType      int    `json:"toType"`
	Text        string `json:"text"`
}

func NewText(text string) *Text {
	return &Text{
		ContentType: ContentTypeText,
		ToType:      1,
		Text:        text,
	}
}

type Callback struct {
	Echo chan *Event
}

func (c Callback) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	if r.Method != "POST" {
		log.Errorf(ctx, "Invalid Method : %s", r.Method)
		http.Error(w, "Method Not Allowed.", http.StatusMethodNotAllowed)
		return
	}
	if strings.Index(r.Header.Get("Content-Type"), "application/json") != 0 {
		log.Errorf(ctx, "Invalid Content-Type : %s", r.Header.Get("Content-Type"))
		http.Error(w, "Bad Request.", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Errorf(ctx, "Can't decode json : %s", err.Error())
		http.Error(w, "Can't decode json", http.StatusInternalServerError)
		return
	}

	for _, res := range req.Result {
		switch res.EventType {
		case EventTypeMessage:
			var msg Message
			err = json.Unmarshal(res.Content, &msg)
			if err != nil {
				log.Errorf(ctx, "Invalid content : %v\n", err)
				http.Error(w, "Invalid content.", http.StatusInternalServerError)
			}
			if msg.ContentType == ContentTypeText {
				log.Infof(ctx, "Text message : %s", msg.Text)

				err = sendEcho([]string{msg.From}, msg.Text, ctx)
				if err != nil {
					log.Errorf(ctx, "%v\n", err)
					http.Error(w, "Send echo error.", http.StatusInternalServerError)
					return
				}
			}
		case EventTypeOperation:
		}
	}

	fmt.Fprintf(w, "Echo OK")
}

func sendEcho(to []string, text string, ctx context.Context) error {
	e := NewEvent(to)
	e.Content = NewText(text)

	resp, err := e.Send(urlfetch.Client(ctx))
	if err != nil {
		return fmt.Errorf("Send echo error : %v\n", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Can't read response : %v\n", err)
	}
	log.Infof(ctx, "Send event sucess : %s\n", body)

	return nil
}

func init() {
	c := &Callback{}
	http.Handle("/callback", c)
}
