package bot

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/kechako/line-bot-client"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
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
		if msg.ContentType != line.ContentTypeText &&
			msg.ContentType != line.ContentTypeImage &&
			msg.ContentType != line.ContentTypeLocation {
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
		path := ""
		queue := ""
		switch msg.ContentType {
		case line.ContentTypeText:
			path = "/echo"
			queue = "echo"
		case line.ContentTypeImage:
			path = "/vision"
			queue = "vision"
		case line.ContentTypeLocation:
			path = "/rainfall"
			queue = "rainfall"
		}
		task := taskqueue.NewPOSTTask(path, url.Values{
			"id": {msg.Id},
		})
		_, err = taskqueue.Add(ctx, task, queue)
		if err != nil {
			log.Errorf(ctx, "Taskqueue error : %s", err.Error())
			http.Error(w, "Taskqueue error", http.StatusInternalServerError)
			return
		}
	}

	fmt.Fprintf(w, "OK")
}
