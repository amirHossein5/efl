package stream

import (
	"fmt"
	"os"
)

func UpdateImage(buf []byte) error {
	f, err := os.Create("image.jpeg")
	if err != nil {
        return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	_, err = f.Write([]byte(buf))
	if err != nil {
        return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}
