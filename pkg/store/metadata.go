package store

import (
	"encoding/json"
)

func SerializeMetadata(m *Metadata) ([]byte, error) {
	return json.Marshal(m)
}

func DeserializeMetadata(data []byte) (*Metadata, error) {
	var m Metadata
	err := json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
