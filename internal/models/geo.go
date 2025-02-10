package models

import (
	"encoding/json"
	"io"
	"os"
)

type GeoData struct {
	Data json.RawMessage `json:"geoData"`
}

func (m *GeoData) Dummy() (map[string]interface{}, error) {
	file, err := os.Open("data/iRISExample.json")
	if err != nil {
			return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
			return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
			return nil, err
	}
	return data, nil
}
