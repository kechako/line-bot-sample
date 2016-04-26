package bot

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/kechako/line-bot-client"
	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

type Callback struct {
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

		err = sendEcho([]string{msg.From}, msg.Text, ctx)
		if err != nil {
			log.Errorf(ctx, "%v\n", err)
			http.Error(w, "Send echo error.", http.StatusInternalServerError)
			return
		}
	}

	fmt.Fprintf(w, "Echo OK")
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
	c := &Callback{}
	http.Handle("/callback", c)
}
