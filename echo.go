package bot

import (
	"io/ioutil"
	"net/http"

	"github.com/kechako/line-bot-client"
	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

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
