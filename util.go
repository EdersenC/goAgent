package goAgent

import (
	"encoding/json"
	"fmt"
	"os"
)
import (
	"io"
)

func ReadJSONFile(file *os.File) ([]byte, error) {
	defer func(path *os.File) {
		err := path.Close()
		if err != nil {
		}
	}(file)
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	return data, nil
}

// BindJSON reads a JSON file and unmarshal it into the provided target
func BindJSON(file *os.File, target interface{}) error {
	data, err := ReadJSONFile(file)

	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}
