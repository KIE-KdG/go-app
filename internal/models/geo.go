package models

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type GeoData struct {
	Data json.RawMessage `json:"geoData"`
}

func (m *GeoData) Dummy() (map[string]interface{}, error) {
	// Try to open the file
	file, err := os.Open("data/iRISExample.json")
	if err != nil {
		// If file doesn't exist, return a fallback GeoJSON
		if os.IsNotExist(err) {
			return createFallbackGeoJSON()
		}
		return nil, fmt.Errorf("error opening GeoJSON file: %w", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading GeoJSON file: %w", err)
	}

	// Ensure we have valid JSON data
	if len(bytes) == 0 {
		return createFallbackGeoJSON()
	}

	var data map[string]interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		// If we can't parse the file, use the fallback
		return createFallbackGeoJSON()
	}
	
	// Verify it has the required GeoJSON structure
	if _, ok := data["type"]; !ok {
		// If it doesn't have a type field, it's not valid GeoJSON
		return createFallbackGeoJSON()
	}
	
	return data, nil
}

// createFallbackGeoJSON generates a simple GeoJSON as fallback if the file is not found
func createFallbackGeoJSON() (map[string]interface{}, error) {
	// A sample GeoJSON FeatureCollection with multiple features
	fallbackJSON := map[string]interface{}{
		"type": "FeatureCollection",
		"features": []map[string]interface{}{
			{
				"type": "Feature",
				"properties": map[string]interface{}{
					"name": "Example Point",
					"type": "point_of_interest",
					"description": "This is a fallback point feature",
				},
				"geometry": map[string]interface{}{
					"type": "Point",
					"coordinates": []float64{0.0, 0.0},
				},
			},
			{
				"type": "Feature",
				"properties": map[string]interface{}{
					"name": "Example Polygon",
					"type": "area",
					"description": "This is a fallback polygon feature",
				},
				"geometry": map[string]interface{}{
					"type": "Polygon",
					"coordinates": [][][]float64{
						{
							{-5.0, -5.0},
							{-5.0, 5.0},
							{5.0, 5.0},
							{5.0, -5.0},
							{-5.0, -5.0},
						},
					},
				},
			},
			{
				"type": "Feature",
				"properties": map[string]interface{}{
					"name": "Example LineString",
					"type": "river",
					"description": "This is a fallback line feature",
				},
				"geometry": map[string]interface{}{
					"type": "LineString",
					"coordinates": [][]float64{
						{-10.0, 0.0},
						{-5.0, 2.0},
						{0.0, 0.0},
						{5.0, -2.0},
						{10.0, 0.0},
					},
				},
			},
		},
	}

	return fallbackJSON, nil
}