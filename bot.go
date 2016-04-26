package bot

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/kechako/line-bot-client"
	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"
)

func callbackHandler(w http.ResponseWriter, r *http.Request) {
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

	dump, err := httputil.DumpRequest(r, true)
	if err == nil {
		log.Infof(ctx, "Request : %s", dump)
	}

	req, err := line.ParseRequest(r.Body)
	if err != nil {
		log.Errorf(ctx, "Can nott parse request : %s", err.Error())
		http.Error(w, "Can not parse request", http.StatusInternalServerError)
		return
	}

	for _, res := range req.Result {
		if res.EventType != line.EventTypeMessage || res.Message == nil {
			continue
		}

		msg := res.Message
		if msg.ContentType != line.ContentTypeText {
			continue
		}

		// Datastore にメッセージを保存
		key := datastore.NewKey(ctx, "LineMessage", msg.Id, 0, nil)
		_, err = datastore.Put(ctx, key, msg)
		if err != nil {
			log.Errorf(ctx, "Datastore put error : %s", err.Error())
			http.Error(w, "Datastore put error", http.StatusInternalServerError)
			return
		}

		// Taskqueue に登録
		task := taskqueue.NewPOSTTask("/echo", url.Values{
			"id": {msg.Id},
		})
		_, err = taskqueue.Add(ctx, task, "echo")
		if err != nil {
			log.Errorf(ctx, "Taskqueue error : %s", err.Error())
			http.Error(w, "Taskqueue error", http.StatusInternalServerError)
			return
		}
	}

	fmt.Fprintf(w, "OK")
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	id := r.FormValue("id")

	if id == "" {
		log.Errorf(ctx, "Id is empty")
		http.Error(w, "Id is empty", http.StatusBadRequest)
		return
	}

	// Datastore からメッセージ取得
	key := datastore.NewKey(ctx, "LineMessage", id, 0, nil)
	msg := &line.Message{}
	err := datastore.Get(ctx, key, msg)
	if err != nil {
		log.Errorf(ctx, "Datastore get error : %s", err.Error())
		http.Error(w, "Datastore get error", http.StatusInternalServerError)
		return
	}

	err = sendEcho([]string{msg.From}, msg.Text, ctx)
	if err != nil {
		log.Errorf(ctx, "%v\n", err)
		http.Error(w, "Send echo error.", http.StatusInternalServerError)
		return
	}
}

func sendEcho(to []string, text string, ctx context.Context) error {
	e := line.NewEvent(to)
	e.Content = line.NewText(text)

	resp, err := e.Send(urlfetch.Client(ctx))
	if err != nil {
		return errors.Wrap(err, "Send event error")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Can not read response")
	}

	log.Infof(ctx, "Send event sucess : %s\n", body)

	return nil
}

func init() {
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/echo", echoHandler)
}
