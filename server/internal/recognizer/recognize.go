package recognizer

import (
	"fmt"
	"log"

	"github.com/Kagami/go-face"
	"github.com/amirhossein5/efl/server/internal/dbconnection"
	"github.com/amirhossein5/efl/server/internal/models"
)

var Rec *face.Recognizer = nil

func RecognizeUser(buf []byte) (int, error) {
	currentFace, err := Rec.RecognizeSingle([]byte(buf))
	if err != nil {
		return -1, fmt.Errorf("failed to recognize given buffer: %v", err)
	}
	if currentFace == nil {
		return -1, fmt.Errorf("not a single face on the image")
	}

	return Rec.ClassifyThreshold(currentFace.Descriptor, 0.2), nil
}

func Initialize() (*face.Recognizer, error) {
	if Rec == nil {
		log.Println("initializing face-recognition-models...")

		var err error
		Rec, err = face.NewRecognizer("face-recognition-models")
		if err != nil {
			return nil, fmt.Errorf("failed to load recognizer: %v", err)
		}

		var enrolledFaces []models.EnrolledFace
		dbconnection.Conn.Find(&enrolledFaces)

		var samples []face.Descriptor
		var userIds []int32

		for _, enrolledFace := range enrolledFaces {
			rface, err := Rec.RecognizeSingleFile(enrolledFace.Path)
			if err != nil {
				log.Printf("Can't recognize: %v, enrolled face: %v\n", err, enrolledFace)
				continue
			}

			samples = append(samples, rface.Descriptor)
			userIds = append(userIds, int32(enrolledFace.UserID))
		}

		Rec.SetSamples(samples, userIds)
	}

	return Rec, nil
}
