package types

import (
	"strings"

	id3 "github.com/mikkyang/id3-go"
	"github.com/zmb3/spotify"
)

type File struct {
	Song
	Filename string
	Dir      string
}

type FileWithSpotifyTrack struct {
	File         File
	SpotifyTrack spotify.FullTrack
}

type Song interface {
	Artist() string
	Title() string
	Genre() string
	Year() string
	SetGenre(string)
	SetYear(string)
	Save() error
}

type ID3Wrapper struct {
	*id3.File
}

type CleanWrapper struct {
	Song
}

func (wrapper CleanWrapper) Artist() string {
	return strings.TrimRight(wrapper.Song.Artist(), "\x00")
}

func (wrapper CleanWrapper) Title() string {
	return strings.TrimRight(wrapper.Song.Title(), "\x00")
}

func (wrapper CleanWrapper) Genre() string {
	return strings.TrimRight(wrapper.Song.Genre(), "\x00")
}

func (file ID3Wrapper) Save() error {
	return file.Close()
}
