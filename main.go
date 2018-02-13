package main

import (
	"context"
	"os"
	"regexp"
	"strings"

	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configstore"
	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/files"
	"github.com/snikch/musicmanager/spotifyclient"
	"github.com/zmb3/spotify"
)

var limit = 50

type SongKey struct {
	Artist string
	Title  string
}

func main() {
	ctx := configuration.ContextWithConfiguration(context.Background())
	defer func() {
		log.Debug("Saving configuration to file")
		conf := configuration.ContextConfiguration(ctx)
		err := configstore.Save(conf)
		if err != nil {
			log.WithError(err).Error("Could not save configuration")
		}
	}()
	ctx, err := spotifyclient.ContextWithClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// getAllMusicFiles(ctx)
	createMissingPlaylist(ctx)
}

func getAllSpotifyTracks(ctx context.Context) (string, map[SongKey]spotify.FullTrack, map[SongKey][]spotify.SimplePlaylist) {
	var userID string
	client := spotifyclient.ContextClient(ctx)
	opts := &spotify.Options{Limit: &limit}
	playlists, err := client.CurrentUsersPlaylistsOpt(opts)
	if err != nil {
		log.Fatal(err)
	}
	conf := configuration.ContextConfiguration(ctx)
	log.WithField("regex", conf.Spotify.PlaylistRegex).Debug("Matching playlists")
	matcher, err := regexp.Compile(conf.Spotify.PlaylistRegex)
	if err != nil {
		log.WithError(err).Fatal()
	}
	trackLists := map[SongKey][]spotify.SimplePlaylist{}
	tracks := map[SongKey]spotify.FullTrack{}

	for _, playlist := range playlists.Playlists {
		if !matcher.Match([]byte(playlist.Name)) {
			log.Debug("Skipping non matching playlist")
			continue
		}
		log.WithField("name", playlist.Name).Info("Processing Playlist")
		var fullList *spotify.PlaylistTrackPage
		for {
			if fullList != nil {
				offset := fullList.Offset + len(fullList.Tracks)
				opts.Offset = &offset
			} else {
				offset := 0
				opts.Offset = &offset
			}
			userID = playlist.Owner.ID
			fullList, err = client.GetPlaylistTracksOpt(playlist.Owner.ID, playlist.ID, opts, "")
			if err != nil {
				log.WithError(err).Fatal()
			}
			if len(fullList.Tracks) == 0 {
				break
			}

			log.WithField("tracks", len(fullList.Tracks)).WithField("offset", *opts.Offset).Debug("Got tracks")
			for _, listTrack := range fullList.Tracks {
				track := listTrack.Track
				artistParts := []string{}
				for _, part := range track.Artists {
					artistParts = append(artistParts, part.Name)
				}
				key := SongKey{
					Artist: strings.Join(artistParts, ", "),
					Title:  track.Name,
				}
				log.WithField("key", key).Debug("Spotify track")
				tracks[key] = track
				trackLists[key] = append(trackLists[key], playlist)
			}
		}
	}
	log.WithField("total", len(tracks)).Info("Found tracks")
	return userID, tracks, trackLists
}
func getAllMusicFiles(ctx context.Context) []*files.File {
	allFiles := []*files.File{}
	conf := configuration.ContextConfiguration(ctx)
	for _, dir := range conf.MusicFiles.Dirs {
		dirFiles, err := files.LoadDir(dir)
		if err != nil {
			log.WithError(err).Fatal()
		}
		allFiles = append(allFiles, dirFiles...)
	}
	log.WithField("total", len(allFiles)).Info("Found music files")
	return allFiles
}

func fileKey(file *files.File) SongKey {
	return SongKey{
		Artist: strings.TrimRight(file.Artist(), "\x00"),
		Title:  strings.TrimRight(file.Title(), "\x00"),
	}
}

func updateFileWithPlaylistTags(ctx context.Context, file *files.File, track []spotify.SimplePlaylist) {
	// log.WithField("title", file.Title()).
	// 	WithField("comments", file.Frame("TCOM")).
	// 	Debug("Processing frames")
	// for _, frames := range file.AllFrames() {
	// 	log.WithField("frame", frames).Debug("Frame")
	// }
}
func createMissingPlaylist(ctx context.Context) {
	userID, tracks, trackLists := getAllSpotifyTracks(ctx)
	allFiles := getAllMusicFiles(ctx)
	for _, file := range allFiles {
		key := fileKey(file)
		if key.Artist == "" || key.Title == "" {
			log.WithField("key", key).
				WithField("name", file.Filename).
				Debug("Skipping unknown track")
			continue
		}
		if track, exists := tracks[key]; exists {
			log.WithField("key", key).WithField("spotify id", track.ID.String()).Debug("Found track")
			updateFileWithPlaylistTags(ctx, file, trackLists[key])
			delete(tracks, key)
		} else {
			log.WithField("key", key).Info("Couldn't find track")
		}
	}

	for key := range tracks {
		log.WithField("key", key).Info("Track missing from music files")
	}
	if len(tracks) == 0 {
		log.Info("Nothing to do")
		os.Exit(1)
	}
	createPlaylistFromTracks(ctx, userID, tracks)
}

func createPlaylistFromTracks(ctx context.Context, userID string, tracks map[SongKey]spotify.FullTrack) {
	conf := configuration.ContextConfiguration(ctx)
	client := spotifyclient.ContextClient(ctx)
	playlistID := spotify.ID(conf.Spotify.OutputPlaylist.ID)
	if playlistID == "" {
		playlist, err := client.CreatePlaylistForUser(userID, conf.Spotify.OutputPlaylist.Name, false)
		log.
			WithField("user", userID).
			WithField("name", conf.Spotify.OutputPlaylist.Name).
			Info("Created new playlist")
		if err != nil {
			log.WithError(err).Fatal()
		}
		playlistID = playlist.ID
		conf.Spotify.OutputPlaylist.ID = string(playlistID)
	} else {
		log.WithField("id", playlistID).Debug("Clearing existing playlist")
		err := client.ReplacePlaylistTracks(userID, playlistID)
		if err != nil {
			log.WithError(err).Fatal()
		}
	}
	batch := []spotify.ID{}
	count := 0
	for _, track := range tracks {
		if track.ID.String() == "" {
			log.WithField("name", track.Name).Warn("Skipping track with no id")
			continue
		}
		count++
		batch = append(batch, track.ID)
		log.
			WithField("id", track.ID).
			WithField("name", track.Name).
			Debug("Adding track to playlist")
		if count >= 100 {
			log.Debug("Saving playlist")
			_, err := client.AddTracksToPlaylist(userID, playlistID, batch...)
			if err != nil {
				log.WithError(err).Fatal()
			}
			batch = []spotify.ID{}
			count = 0
		}
	}
	if len(batch) > 0 {
		_, err := client.AddTracksToPlaylist(userID, playlistID, batch...)
		if err != nil {
			log.WithError(err).Fatal()
		}
	}
	log.WithField("id", playlistID).Info("Playlist updated")
}

func main2() {
	// 	file, err := os.Open("/Users/mal/Music/iTunes/iTunes Media/Music/Alex Metric/Ammunition Pt. 4 (Remixes)/1-03 Always There - Purple Disco Machine Remix.mp3")
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	tags, err := tag.ReadFrom(file)
	// 	if err != nil {
	// 		// log.Fatal(err)
	// 	}
	// 	for name, content := range tags.Raw() {
	// 		// log.Printf("%s: %v\n", name, content)
	// 	}
	// 	// log.Println(string(tags.Raw()["GP1"].([]uint8)))

	// 	saver, err := id3.Open("/Users/mal/Music/iTunes/iTunes Media/Music/Alex Metric/Ammunition Pt. 4 (Remixes)/1-03 Always There - Purple Disco Machine Remix.mp3")
	// 	if err != nil {
	// 		// log.Fatal(err)
	// 	}
	// 	f := saver.Frame("TBP")
	// 	// log.Println(f)
	// 	// log.Println(saver.Version())
	// 	frame := v2.NewTextFrame(v2.V22FrameTypeMap["GP1"], "test")
	// 	saver.AddFrames(frame)
	// 	err = ioutil.WriteFile("test.mp3", saver.Bytes(), 0644)
	// 	if err != nil {
	// 		// log.Fatal(err)
	// 	}
}
