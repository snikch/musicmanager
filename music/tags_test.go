package music

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/types"
	"github.com/zmb3/spotify"
)

type MockSong struct {
	comment string
	genre   string
	year    string
	title   string
	artist  string
}

func (f *MockSong) Artist() string {
	return f.artist
}

func (f *MockSong) Comment() string {
	return f.comment
}

func (f *MockSong) Genre() string {
	return f.genre
}

func (f *MockSong) Year() string {
	return f.year
}

func (f *MockSong) Title() string {
	return f.title
}

func (f *MockSong) SetComment(comment string) {
	f.comment = comment
}

func (f *MockSong) SetGenre(genre string) {
	f.genre = genre
}

func (f *MockSong) SetYear(year string) {
	f.year = year
}

func (f *MockSong) Save() error {
	return nil
}

func newMockFile(artist, title string) types.File {
	return types.File{
		Song: &MockSong{
			title:  title,
			artist: artist,
		},
	}
}

func TestUpdateGenreAddsTags(t *testing.T) {
	ctx := configuration.ContextWithConfiguration(context.Background())
	song := newMockFile("artist", "song")
	updateGenre(ctx, log.WithField("test", nil), types.FileWithContext{File: song, SpotifyPlaylists: []spotify.SimplePlaylist{
		{Name: "P1 P2"},
		{Name: "P1 P3"},
	}})
	tagMatch(t, song.Genre(), "p1 p2 p3")
}

func TestUpdateGenreLeavesTags(t *testing.T) {
	ctx := configuration.ContextWithConfiguration(context.Background())
	song := newMockFile("artist", "song")
	song.SetGenre("x1")
	updateGenre(ctx, log.WithField("test", nil), types.FileWithContext{File: song, SpotifyPlaylists: []spotify.SimplePlaylist{
		{Name: "P1 P2"},
	}})
	tagMatch(t, song.Genre(), "x1 p1 p2")
}

func TestUpdateGenreRemovesTags(t *testing.T) {
	ctx := configuration.ContextWithConfiguration(context.Background())
	conf := configuration.ContextConfiguration(ctx)
	conf.MusicFiles.TagRemovals = []string{"p1", "x1"}
	song := newMockFile("artist", "song")
	song.SetGenre("x1")
	updateGenre(ctx, log.WithField("test", nil), types.FileWithContext{File: song, SpotifyPlaylists: []spotify.SimplePlaylist{
		{Name: "P1 P2"},
	}})
	tagMatch(t, song.Genre(), "p2")
}

func TestUpdateGenreRemovesTagsWithNoUpdate(t *testing.T) {
	ctx := configuration.ContextWithConfiguration(context.Background())
	conf := configuration.ContextConfiguration(ctx)
	conf.MusicFiles.TagRemovals = []string{"x1"}
	song := newMockFile("artist", "song")
	song.SetGenre("x1 p1")
	updateGenre(ctx, log.WithField("test", nil), types.FileWithContext{File: song, SpotifyPlaylists: []spotify.SimplePlaylist{
		{Name: "P1"},
	}})
	tagMatch(t, song.Genre(), "p1")
}

func TestUpdateGenreReplacesTags(t *testing.T) {
	ctx := configuration.ContextWithConfiguration(context.Background())
	conf := configuration.ContextConfiguration(ctx)
	conf.MusicFiles.TagReplacements = map[string]string{
		"P1 One":  "P1One",
		"P1 None": "",
		":":       "",
	}
	song := newMockFile("artist", "song")
	updateGenre(ctx, log.WithField("test", nil), types.FileWithContext{File: song, SpotifyPlaylists: []spotify.SimplePlaylist{
		{Name: "P1 One: P2"},
		{Name: "P1 None: P3"},
	}})
	tagMatch(t, song.Genre(), "p1one p2 p3")
}

func tagMatch(t *testing.T, a, b string) {
	aSlice := strings.Split(a, " ")
	bSlice := strings.Split(b, " ")
	sort.Strings(aSlice)
	sort.Strings(bSlice)
	if !reflect.DeepEqual(aSlice, bSlice) {
		t.Fatalf("Expected '%s' genre but got '%s'", b, a)
	}
}
