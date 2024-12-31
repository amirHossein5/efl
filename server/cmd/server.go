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
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func cameraWebsocketHandler(ws *websocket.Conn) {
	for {
		var buf string
		err := websocket.Message.Receive(ws, &buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("failed to read websocket data", err)
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
	w.Header().Add("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	boundary := "\r\n--frame\r\nContent-Type: image/jpeg\r\n\r\n"

	for {
		n, err := io.WriteString(w, boundary)
		if err != nil || n != len(boundary) {
			return
		}

		f, err := os.Open("image.jpeg")
		if err != nil {
			return
		}

		_, err = f.WriteTo(w)
		if err != nil {
			return
		}

		n, err = io.WriteString(w, "\r\n")
		if err != nil || n != 2 {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}
}
