package ping

import (
	"encoding/json"
)

func resultData(data []byte, destination any) error {
	return json.Unmarshal(data, destination)
}
