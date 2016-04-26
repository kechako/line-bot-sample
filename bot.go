package bot

import "net/http"

func init() {
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/echo", echoHandler)
	http.HandleFunc("/vision", visionHandler)
}
