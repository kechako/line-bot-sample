package bot

import (
	"io"
	"net/http"
	"os"

	"github.com/kechako/line-bot-client"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/cloud/storage"
)

var (
	BucketName = os.Getenv("BUCKET_NAME")
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

	// Cloud Storage に格納
	bucketName := BucketName
	if bucketName == "" {
		bucketName, err = file.DefaultBucketName(ctx)
		if err != nil {
			log.Errorf(ctx, "Can not get default bucket name : %v\n", err)
			http.Error(w, "Can not get default bucket name.", http.StatusInternalServerError)
			return
		}
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorf(ctx, "Can not create storage client : %v\n", err)
		http.Error(w, "Can not create storage client.", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	writer := client.Bucket(bucketName).Object(msg.Id).NewWriter(ctx)
	defer writer.Close()

	size, err := io.Copy(writer, res.Body)
	if err != nil {
		log.Errorf(ctx, "Storage write error : %v\n", err)
		http.Error(w, "Storage write error.", http.StatusInternalServerError)
		return
	}

	log.Infof(ctx, "Write to storage : %d", size)
}
