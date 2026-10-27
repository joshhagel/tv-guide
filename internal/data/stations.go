package data

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed spokane-stations.json
var SpokaneStationsJSON []byte

type OverlayStation struct {
	Callsign    string `json:"callsign"`
	Name        string `json:"name"`
	GuidePrefix string `json:"guidePrefix"`
	Network     string `json:"network"`
	ExactOnly   bool   `json:"exactOnly"`
}

func LoadOverlayStations() ([]OverlayStation, error) {
	var stations []OverlayStation
	if err := json.Unmarshal(SpokaneStationsJSON, &stations); err != nil {
		return nil, fmt.Errorf("parse spokane stations: %w", err)
	}
	return stations, nil
}
