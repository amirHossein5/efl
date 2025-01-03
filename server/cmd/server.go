package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/amirhossein5/efl/server/internal/dbconnection"
	"github.com/amirhossein5/efl/server/internal/models"
	"github.com/amirhossein5/efl/server/internal/recognizer"
	"github.com/amirhossein5/efl/server/internal/stream"
	"github.com/gorilla/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const PLAY_SOUND_WARNING = "play-sound:warning"
const PLAY_SOUND_SUCCESS = "play-sound:success"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Accepting all requests
	},
	ReadBufferSize:  1024 * 1024,
	WriteBufferSize: 1024 * 1024,
}

func main() {
	db, err := dbconnection.Open(sqlite.Open("database.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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

	rec, err := recognizer.Initialize()
	if err != nil {
		log.Fatalf("failed to initialize recognizer: %v", err)
	}
	defer rec.Close()

	http.HandleFunc("/", indexPage)
	http.HandleFunc("/stream", streamHandler)
	http.HandleFunc("/camera-websocket", cameraWebsocketHandler)

	log.Println("starting webserver at :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func cameraWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	for {
		time.Sleep(100 * time.Millisecond)

		mt, buf, err := c.ReadMessage()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("failed to read websocket data", err)
			c.Close()
			break
		}
		if mt != 2 {
			continue
		}

		err = stream.UpdateImage(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		userId, err := recognizer.RecognizeUser(buf)
		if err != nil {
			log.Println(err)
			continue
		}
		if userId <= 0 {
			log.Println("Can't classify")
			err = c.WriteMessage(mt, []byte(PLAY_SOUND_WARNING))
			if err != nil {
				log.Println("send failed ", err)
				continue
			}
			continue
		}

		log.Println(userId)

		var user models.User
		dbconnection.Conn.Find(&user, userId)

		can, latestAttendanceLog, err := user.CanLogAttendance()
		if err != nil {
			log.Println("failed to CanLogAttendance:", err)
			continue
		}
		if !can {
			if time.Now().Add(-5 * time.Second).After(latestAttendanceLog.CreatedAt) {
				err = c.WriteMessage(mt, []byte(PLAY_SOUND_WARNING))
				if err != nil {
					log.Println("send failed ", err)
					continue
				}
			}
			continue
		}

		err = user.LogAttendance()
		if err != nil {
			log.Println("failed to LogAttendance:", err)
			continue
		}

		err = c.WriteMessage(mt, []byte(PLAY_SOUND_SUCCESS))
		if err != nil {
			log.Println("send failed ", err)
			continue
		}
	}
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	const tmplFile = "views/index.html"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		log.Printf("failed to parse file %v, %v\n", tmplFile, err)
		return
	}

	var attendanceLogs []models.AttendanceLog
	err = dbconnection.Conn.Where("DATE(created_at) = ?", time.Now().Format("2006-01-02")).Order("created_at DESC").Preload("User").Find(&attendanceLogs).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		log.Printf("failed to fetch attendanceLogs %v\n", err)
		return
	}

	err = tmpl.Execute(w, attendanceLogs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		log.Printf("failed to execute file %v, %v\n", tmplFile, err)
		return
	}
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
