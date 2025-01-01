package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Kagami/go-face"
	"github.com/amirhossein5/efl/server/internal/dbconnection"
	"github.com/amirhossein5/efl/server/internal/models"
	"golang.org/x/net/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := dbconnection.Open(sqlite.Open("database.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatal(err, " in users table")
	}
	err = db.AutoMigrate(&models.EnrolledFace{})
	if err != nil {
		log.Fatal(err, " in enrolled_faces table")
	}
	err = db.AutoMigrate(&models.AttendanceLog{})
	if err != nil {
		log.Fatal(err, " in attendance_logs table")
	}

	http.HandleFunc("/", indexPage)
	http.HandleFunc("/stream", streamHandler)
	http.Handle("/camera-websocket", websocket.Handler(cameraWebsocketHandler))

	log.Println("starting webserver at :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func cameraWebsocketHandler(ws *websocket.Conn) {
	for {
		time.Sleep(100 * time.Millisecond)

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

		rec, err := face.NewRecognizer("face-recognition-models")
		if err != nil {
			log.Printf("Can't init face recognizer: %v\n", err)
			continue
		}
		defer rec.Close()

		var enrolledFaces []models.EnrolledFace
		dbconnection.Conn.Find(&enrolledFaces)

		var samples []face.Descriptor
		var cats []int32

		for _, enrolledFace := range enrolledFaces {
			rface, err := rec.RecognizeSingleFile(enrolledFace.Path)
			if err != nil {
				log.Printf("Can't recognize: %v, enrolled face: %v\n", err, enrolledFace)
				continue
			}

			samples = append(samples, rface.Descriptor)
			cats = append(cats, int32(enrolledFace.UserID))
		}

		rec.SetSamples(samples, cats)

		currentFace, err := rec.RecognizeSingle([]byte(buf))
		if err != nil {
			log.Printf("Can't recognize: %v\n", err)
			continue
		}
		if currentFace == nil {
			log.Printf("Not a single face on the image\n")
			continue
		}

		userId := rec.Classify(currentFace.Descriptor)
		if userId < 0 {
			log.Println("Can't classify")
			websocket.Message.Send(ws, "play-sound:warning")
			continue
		}

		log.Println(userId)

		attendanceLog := models.AttendanceLog{UserID: uint64(userId)}
		dbconnection.Conn.Create(attendanceLog)

		websocket.Message.Send(ws, "play-sound:success")
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
