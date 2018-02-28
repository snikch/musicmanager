package types

import (
	"encoding/json"
	"errors"
)

type SongKey struct {
	Artist string
	Title  string
}

func (key SongKey) MarshalText() ([]byte, error) {
	return json.Marshal([]string{key.Artist, key.Title})
}

func (key *SongKey) UnmarshalText(content []byte) error {
	parts := []string{}
	err := json.Unmarshal(content, &parts)
	if err != nil {
		return err
	}
	if len(parts) != 2 {
		return errors.New("SongKey.UnmarshalJSON: Requires two parts")
	}
	key.Artist = parts[0]
	key.Title = parts[1]
	return nil
}
