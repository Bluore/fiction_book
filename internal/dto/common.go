package dto

import (
	"encoding/json"
	"time"
)

type UnixTime time.Time

func (t *UnixTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(*t).Unix())
}

func (t *UnixTime) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return err
	}
	*t = UnixTime(time.Unix(timestamp, 0))
	return nil
}
