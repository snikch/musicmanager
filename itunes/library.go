package itunes

// This file is copied from https://github.com/ericdaugherty/itunesexport-go
// All copyright belongs to the original author.

import (
	"os"
	"strconv"
	"time"

	plist "github.com/DHowett/go-plist"
)

type Library struct {
	MajorVersion        int `plist:"Major Version"`
	MinorVersion        int `plist:"Minor Version"`
	Date                time.Time
	ApplicationVersion  int
	Features            int
	ShowContentRating   bool   `plist:"Show Content Ratings"`
	MusicFolder         string `plist:"Music Folder"`
	LibraryPersistentID string `plist:"Library Persistent ID"`
	Tracks              map[string]Track
	Playlists           []Playlist
	PlaylistMap         map[string]Playlist
}

type Track struct {
	TrackID             int `plist:"Track ID"`
	Name                string
	Artist              string
	AlbumArtist         string `plist:"Album Artist"`
	Composer            string
	Album               string
	Genre               string
	Kind                string
	Size                int
	TotalTime           int `plist:"Total Time"`
	TrackNumber         int `plist:"Track Number"`
	Year                int
	DateModified        time.Time `plist:"Date Modified"`
	DateAdded           time.Time `plist:"Date Added"`
	BitRate             int       `plist:"Bit Rate"`
	SampleRate          int       `plist:"Sample Rate"`
	PlayCount           int       `plist:"Play Count"`
	PlayDate            int       `plist:"Play Date"`
	PlayDateUTC         time.Time `plist:"Play Date UTC"`
	SkipCount           int       `plist:"Skip Count"`
	SkipDate            time.Time `plist:"Skip Date"`
	Rating              int
	AlbumRating         int    `plist:"Album Rating"`
	AlbumRatingComputed bool   `plist:"Album Rating Computed"`
	ArtworkCount        int    `plist:"Artwork Count"`
	PersistentID        string `plist:"Persistent ID"`
	TrackType           string `plist:"Track Type"`
	Location            string
	FileFolderCount     int `plist:"File Folder Count"`
	LibraryFolderCount  int `plist:"Library Folder Count"`
}

type Playlist struct {
	Name                 string
	Master               bool
	PlaylistID           int    `plist:"Playlist ID"`
	PlaylistPersistentID string `plist:"Playlist Persistent ID"`
	DistinguishedKind    int    `plist:"Distinguished Kind"`
	Visible              bool
	AllItems             bool           `plist:"All Items"`
	SmartInfo            []byte         `plist:"Smart Info"`
	SmartCriteria        []byte         `plist:"Smart Criteria"`
	PlaylistItems        []PlaylistItem `plist:"Playlist Items"`
}

type PlaylistItem struct {
	TrackID int `plist:"Track ID"`
}

func LoadLibrary(fileLocation string) (returnLibrary *Library, err error) {

	if _, statErr := os.Stat(fileLocation); os.IsNotExist(statErr) {
		err = statErr
		return
	}

	file, pathErr := os.Open(fileLocation)
	if pathErr != nil {
		err = pathErr
		return
	}

	decoder := plist.NewDecoder(file)

	var library Library
	decodeErr := decoder.Decode(&library)
	if decodeErr != nil {
		err = decodeErr
		return
	}

	library.PlaylistMap = make(map[string]Playlist)
	for _, value := range library.Playlists {
		library.PlaylistMap[value.Name] = value
	}

	return &library, err
}

func (playlist *Playlist) Tracks(library *Library) (tracks []Track) {
	for _, item := range playlist.PlaylistItems {
		track, ok := library.Tracks[strconv.FormatInt(int64(item.TrackID), 10)]
		if ok {
			tracks = append(tracks, track)
		}
	}
	return
}
