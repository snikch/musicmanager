package types

import (
	"strings"

	"github.com/bogem/id3v2"
	id3 "github.com/mikkyang/id3-go"
	"github.com/snikch/musicmanager/itunes"
	"github.com/zmb3/spotify"
)

type File struct {
	Song
	Filename string
	Dir      string
}

type FileContexts map[SongKey]FileWithContext
type FileWithContext struct {
	File
	SpotifyTrack     *spotify.FullTrack
	SpotifyPlaylists []spotify.SimplePlaylist
	ITunesTrack      *itunes.Track
}

type Song interface {
	Artist() string
	Title() string
	Comment() string
	Genre() string
	Year() string
	SetComment(string)
	SetGenre(string)
	SetYear(string)
	Save() error
}

type ID3Wrapper struct {
	*id3.File
}

func (file ID3Wrapper) Comment() string {
	comments := file.File.Comments()
	if len(comments) == 0 {
		return ""
	}
	return strings.Join(comments, "\n")
}
func (file ID3Wrapper) SetComment(comment string) {
	panic("Cannot set comment")
}

func (file ID3Wrapper) Save() error {
	return file.Close()
}

type ID3V2Wrapper struct {
	*id3v2.Tag
}

func (tag ID3V2Wrapper) Comment() string {
	comments := tag.Tag.GetFrames(tag.Tag.CommonID("Comments"))
	if len(comments) == 0 {
		return ""
	}
	var value string
	for _, frame := range comments {
		f := frame.(id3v2.CommentFrame)
		if f.Language != "eng" {
			continue
		}
		// log.WithField("Frame", f).Info("frame")
		value = value + f.Text
	}
	return value
}

func (tag ID3V2Wrapper) SetComment(comment string) {
	tag.Tag.DeleteFrames(tag.Tag.CommonID("Comments"))
	tag.Tag.AddCommentFrame(id3v2.CommentFrame{
		Encoding: id3v2.EncodingISO,
		// Encoding: id3v2.EncodingUTF8,
		Language: "eng",
		Text:     comment,
	})
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

func (wrapper CleanWrapper) Comment() string {
	return strings.TrimRight(wrapper.Song.Comment(), "\x00")
}
