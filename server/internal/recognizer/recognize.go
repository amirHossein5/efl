package recognizer

import (
	"fmt"
	"log"

	"github.com/Kagami/go-face"
	"github.com/amirhossein5/efl/server/internal/dbconnection"
	"github.com/amirhossein5/efl/server/internal/models"
)

var rec *face.Recognizer = nil

func RecognizeUser(buf []byte) (uint64, error) {
	if rec == nil {
		log.Println("initializing face-recognition-models...")
		var err error
		rec, err = face.NewRecognizer("face-recognition-models")
		if err != nil {
			return 0, fmt.Errorf("failed to load recognizer: %v", err)
		}
	}
	// defer rec.Close()

	var enrolledFaces []models.EnrolledFace
	dbconnection.Conn.Find(&enrolledFaces)

	var samples []face.Descriptor
	var userIds []int32

	for _, enrolledFace := range enrolledFaces {
		rface, err := rec.RecognizeSingleFile(enrolledFace.Path)
		if err != nil {
			log.Printf("Can't recognize: %v, enrolled face: %v\n", err, enrolledFace)
			continue
		}

		samples = append(samples, rface.Descriptor)
		userIds = append(userIds, int32(enrolledFace.UserID))
	}

	rec.SetSamples(samples, userIds)

	currentFace, err := rec.RecognizeSingle([]byte(buf))
	if err != nil {
		return 0, fmt.Errorf("failed to recognize given buffer: %v", err)
	}
	if currentFace == nil {
		return 0, fmt.Errorf("not a single face on the image")
	}

	return uint64(rec.Classify(currentFace.Descriptor)), nil
}
