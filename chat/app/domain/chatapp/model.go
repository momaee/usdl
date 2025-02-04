package chatapp

import "encoding/json"

type status struct {
	Status string `json:"status"`
}

// Encode implements the encoder interface.
func (app status) Encode() ([]byte, string, error) {
	data, err := json.Marshal(app)
	return data, "application/json", err
}
