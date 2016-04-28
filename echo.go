package bot

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/kechako/line-bot-client"
	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

const (
	zun     = "ズン"
	doko    = "ドコ"
	kiyoshi = "キ・ヨ・シ！"
)

var (
	keywords = []string{"zundoko", "ズンドコ", "ずんどこ"}
	random   = rand.New(rand.NewSource(time.Now().UnixNano()))
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

	text := msg.Text
	if isZundoko(text) {
		text = zundoko()
	}

	err = sendEcho([]string{msg.From}, text, ctx)
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

func isZundoko(text string) bool {
	for _, word := range keywords {
		if strings.Contains(text, word) {
			return true
		}
	}

	return false
}

func zundoko() string {
	zundoko := [2]string{zun, doko}
	good := [5]string{zun, zun, zun, zun, doko}

	var current [5]string

	reply := bytes.NewBuffer(make([]byte, 0, 1024))

	for current != good {
		shift(&current)
		zd := zundoko[random.Intn(2)]
		current[4] = zd
		reply.WriteString(zd)
	}

	reply.WriteString(kiyoshi)

	return reply.String()
}

func shift(a *[5]string) {
	a[0], a[1], a[2], a[3], a[4] = a[1], a[2], a[3], a[4], ""
}
