package json

import (
	"encoding/json"

	"github.com/pelletier/go-toml/v2"
)

// FromTOML convert toml to json
func FromTOML(v []byte) ([]byte, error) {
	m1 := make(map[interface{}]interface{})
	if err := toml.Unmarshal(v, &m1); err != nil {
		return nil, err
	}
	m2 := convert(m1)
	j, err := json.Marshal(m2)
	if err != nil {
		return nil, err
	}
	return j, nil
}
