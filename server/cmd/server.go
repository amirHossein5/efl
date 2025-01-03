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
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Accepting all requests
	},
	ReadBufferSize:  1024 * 1024,
	WriteBufferSize: 1024 * 1024,
}

var enrollImageForUser *models.User

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

	r := chi.NewRouter()
	r.Get("/", indexPage)
	r.Get("/stream", streamHandler)
	r.Get("/camera-websocket", cameraWebsocketHandler)

	r.Get("/users/create", userCreateHandler)
	r.Post("/users/create", userStoreHandler)

	r.Get("/users/{user_id}/show", userShowHandler)
	r.Post("/users/{user_id}/enroll-face", enrollFaceHandler)

	log.Println("starting webserver at :8000")
	log.Fatal(http.ListenAndServe(":8000", r))
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
			err = c.WriteMessage(mt, []byte(warningMessage("cant classify")))
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
				err = c.WriteMessage(mt, []byte(warningMessage("already logged")))
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

		err = c.WriteMessage(mt, []byte(successMessage("salam "+user.Name)))
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

func userCreateHandler(w http.ResponseWriter, r *http.Request) {
}

func userStoreHandler(w http.ResponseWriter, r *http.Request) {
}

func userShowHandler(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "user_id")

	const tmplFile = "views/users/show.html"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		log.Printf("failed to parse file %v, %v\n", tmplFile, err)
		return
	}

	var user models.User
	err = dbconnection.Conn.Model(&models.User{}).Preload("EnrolledFaces").Find(&user, userId).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		log.Printf("failed to fetch user %v\n", err)
		return
	}

	err = tmpl.Execute(w, user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		log.Printf("failed to execute file %v, %v\n", tmplFile, err)
		return
	}
}

func enrollFaceHandler(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "user_id")

	var user *models.User
	err := dbconnection.Conn.Find(&user, userId).Error
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		log.Printf("enrollFaceHandler: failed to find user: %v\n", err)
		return
	}

	data, err := os.ReadFile("image.jpeg")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		log.Printf("enrollFaceHandler: failed to read image.jpeg file: %v\n", err)
		return
	}

	currentFace, err := recognizer.Rec.RecognizeSingle(data)
	if err != nil || currentFace == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("not a single face on the image"))
		return
	}

	dst := "enrolled-faces/" + uuid.New().String() + ".jpeg"

	err = os.WriteFile(dst, data, 0644)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		log.Printf("enrollFaceHandler: failed to write image to %v: %v\n", dst, err)
		return
	}

	dbconnection.Conn.Create(models.EnrolledFace{UserID: uint64(user.ID), Path: dst})

	http.Redirect(w, r, r.Header.Get("Referer"), 302)
}

func successMessage(msg string) string {
	return "success-message:" + msg
}

func warningMessage(msg string) string {
	return "warning-message:" + msg
}
