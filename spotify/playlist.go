package spotify

import (
	"context"
	"strings"

	"github.com/snikch/api/fail"
	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/spotifyclient"
	"github.com/zmb3/spotify"
)

func CreateMissingPlaylist(ctx context.Context, graph TrackGraph, tracks TrackLookup) error {
	if len(tracks) == 0 {
		log.Info("Nothing to do")
		return nil
	}
	conf := configuration.ContextConfiguration(ctx)
	client := spotifyclient.ContextClient(ctx)
	playlistID := spotify.ID(conf.Spotify.OutputPlaylist.ID)
	if playlistID == "" {
		playlist, err := client.CreatePlaylistForUser(graph.UserID, conf.Spotify.OutputPlaylist.Name, false)
		if err != nil {
			return fail.Trace(err)
		}
		log.
			WithField("user", graph.UserID).
			WithField("name", conf.Spotify.OutputPlaylist.Name).
			Info("Created new spotify missing playlist")
		playlistID = playlist.ID
		conf.Spotify.OutputPlaylist.ID = string(playlistID)
	} else {
		log.WithField("id", playlistID).Info("Clearing existing spotify missing playlist")
		err := client.ReplacePlaylistTracks(graph.UserID, playlistID)
		if err != nil {
			log.WithError(fail.Trace(err)).Fatal()
		}
	}
	batch := []spotify.ID{}
	count := 0
	tracksAdded := 0
	tracksSkipped := 0
	for key, track := range tracks {
		if track.ID.String() == "" {
			log.WithField("name", track.Name).Warn("Skipping track with no id")
			tracksSkipped++
			continue
		}
		tracksAdded++
		count++
		batch = append(batch, track.ID)
		log.
			WithField("id", track.ID).
			WithField("name", track.Name).
			WithField("artist", strings.Join(artistNames(track.Artists), ", ")).
			WithField("playlists", playlistNames(graph.Playlists[key])).
			Info("Adding track to playlist")
		if count >= 100 {
			log.Debug("Saving playlist")
			_, err := client.AddTracksToPlaylist(graph.UserID, playlistID, batch...)
			if err != nil {
				return fail.Trace(err)
			}
			batch = []spotify.ID{}
			count = 0
		}
	}
	if len(batch) > 0 {
		_, err := client.AddTracksToPlaylist(graph.UserID, playlistID, batch...)
		if err != nil {
			return fail.Trace(err)
		}
	}
	log.WithField("id", playlistID).
		WithField("songs", tracksAdded).
		WithField("skipped", tracksSkipped).
		Info("Playlist updated")
	return nil
}

func artistNames(artists []spotify.SimpleArtist) []string {
	names := make([]string, len(artists))
	for i := range artists {
		names[i] = artists[i].Name
	}
	return names
}

func playlistNames(playlists []spotify.SimplePlaylist) []string {
	names := make([]string, len(playlists))
	for i := range playlists {
		names[i] = playlists[i].Name
	}
	return names
}
