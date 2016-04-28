package bot

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/kechako/line-bot-client"
	"github.com/kechako/yolp"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

var (
	yahooAppId = os.Getenv("YAHOO_APP_ID")
)

func init() {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		loc = time.FixedZone("Asia/Tokyo", 9*60*60)
	}

	time.Local = loc
}

func rainfallHandler(w http.ResponseWriter, r *http.Request) {
	if yahooAppId == "" {
		return
	}

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

	text := getRainfallMessage(ctx, msg.Location)

	err = sendEcho([]string{msg.From}, text, ctx)
	if err != nil {
		log.Errorf(ctx, "%v\n", err)
		http.Error(w, "Send echo error.", http.StatusInternalServerError)
		return
	}
}

func getRainfallMessage(ctx context.Context, loc line.Location) string {
	weathers, err := getWeathers(ctx, loc.Latitude, loc.Longitude)
	if err != nil {
		return fmt.Sprintf("取得失敗 : %v", err)
	}

	messages := make([]string, 0, 10)

	// 直近の天気情報
	w := getMostRecentWeather(weathers)

	var result string
	if w.IsRaining() {
		if w.IsObservation() {
			result = "雨降ってます"
		} else {
			result = "雨降ってるかも"
		}
	} else {
		if w.IsObservation() {
			result = "雨降ってないです"
		} else {
			result = "雨降ってないかも"
		}
	}
	messages = append(messages, result+"  "+GetWeatherString(w))

	for _, w := range weathers {
		messages = append(messages, GetWeatherString(w))
	}

	return strings.Join(messages, "\n")
}

func getWeathers(ctx context.Context, latitude, longitude float32) (weathers []yolp.Weather, err error) {
	y := yolp.NewYOLPWithClient(yahooAppId, urlfetch.Client(ctx))

	ydf, err := y.Place(latitude, longitude)
	if err != nil {
		return
	}

	if len(ydf.Feature) == 0 {
		err = fmt.Errorf("Could not get the weather data from the API response.")
		return
	}

	weathers = ydf.Feature[0].Property.WeatherList.Weather
	if len(weathers) == 0 {
		err = fmt.Errorf("Could not get the weather data from the API response.")
		return
	}

	return
}

func getMostRecentWeather(weathers []yolp.Weather) (weather yolp.Weather) {
	now := time.Now()

	var minDuration int64
	for i, w := range weathers {
		d := Abs64(int64(now.Sub(w.Time())))
		if i == 0 || d < minDuration {
			minDuration = d
			weather = w
		}
	}

	return
}

func Abs64(n int64) int64 {
	if n < 0 {
		return -n
	}

	return n
}

func GetWeatherString(w yolp.Weather) string {
	str := fmt.Sprintf("[%s]  %.2f mm", w.Time().Format("15:04"), w.Rainfall)
	if w.IsObservation() {
		return str + "  (実測値)"
	} else if w.IsForecast() {
		return str + "  (予測値)"
	}
	return str
}
