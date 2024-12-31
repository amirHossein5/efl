package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	http.HandleFunc("/", indexPage)
	http.HandleFunc("/stream", streamHandler)
	http.Handle("/camera-websocket", websocket.Handler(cameraWebsocketHandler))

	log.Println("starting webserver at :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("failed to start server", err)
	}
}

func cameraWebsocketHandler(ws *websocket.Conn) {
	for {
		var buf string
		err := websocket.Message.Receive(ws, &buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("failed to read websocket data")
			continue
		}

		f, err := os.Create("image.jpeg")
		if err != nil {
			log.Println("couldn't create file", err)
			continue
		}
		defer f.Close()

		_, err = f.Write([]byte(buf))
		if err != nil {
			log.Println("image write failed", err)
			continue
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "image.jpeg")
}
