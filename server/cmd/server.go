package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	http.Handle("/camera-websocket", websocket.Handler(cameraWebsocketHandler))

	fmt.Println("starting webserver at :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("failed to start server", err)
	}
}

var imageCounter = 0

func cameraWebsocketHandler(ws *websocket.Conn) {
	for {
		buf := make([]byte, ws.Len())
		n, err := ws.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("failed to read websocket data")
			continue
		}

		f, err := os.Create(fmt.Sprint(imageCounter, ".jpeg"))
		if err != nil {
			log.Println("couldn't create file", err)
			continue
		}
		defer f.Close()

		_, err = f.Write(buf[:n])
		if err != nil {
			log.Println("image write failed", err)
			continue
		}

		imageCounter++
		time.Sleep(time.Second)
	}
}

func getFileName() string {
	return time.Now().Format(time.RFC850) + randomString(10) + "-image.jpeg"
}

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
