package bot

import (
	"encoding/base64"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/pkg/errors"

	"github.com/kechako/line-bot-client"

	"google.golang.org/api/vision/v1"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func visionHandler(w http.ResponseWriter, r *http.Request) {
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

	// コンテンツ取得
	res, err := msg.GetContent(urlfetch.Client(ctx))
	if err != nil {
		log.Errorf(ctx, "Content get error : %v\n", err)
		http.Error(w, "Content get error.", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	image, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf(ctx, "Image read error : %v\n", err)
		http.Error(w, "Image read error.", http.StatusInternalServerError)
		return
	}

	batchRes, err := requestVision(ctx, image)
	if err != nil {
		log.Errorf(ctx, "Vision API request error : %v\n", err)
		http.Error(w, "Vision API request error.", http.StatusInternalServerError)
		return
	}

	if len(batchRes.Responses) == 0 {
		log.Infof(ctx, "Vision API did not response")
		return
	}

	err = sendEcho([]string{msg.From}, batchRes.Responses[0].LabelAnnotations[0].Description, ctx)
	if err != nil {
		log.Errorf(ctx, "%v\n", err)
		http.Error(w, "Send echo error.", http.StatusInternalServerError)
		return
	}
}

func requestVision(ctx context.Context, data []byte) (*vision.BatchAnnotateImagesResponse, error) {
	srv, err := vision.New(getClient(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "Can not create the Vision API service")
	}

	image := &vision.Image{
		Content: base64.StdEncoding.EncodeToString(data),
	}

	feature := &vision.Feature{
		Type:       "LABEL_DETECTION",
		MaxResults: 1,
	}

	req := &vision.AnnotateImageRequest{
		Image:    image,
		Features: []*vision.Feature{feature},
	}

	batch := &vision.BatchAnnotateImagesRequest{
		Requests: []*vision.AnnotateImageRequest{req},
	}

	return srv.Images.Annotate(batch).Do()
}

func getClient(ctx context.Context) *http.Client {
	return &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(ctx, vision.CloudPlatformScope),
			Base:   &urlfetch.Transport{Context: ctx},
		},
	}
}
