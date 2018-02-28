package spotify

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/snikch/api/fail"
	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/spotifyclient"
	"github.com/snikch/musicmanager/types"
	"github.com/zmb3/spotify"
)

const (
	playlistCacheLoc = ".cache/playlists"
)

var (
	limit = 50
)

type TrackLookup map[types.SongKey]spotify.FullTrack
type PlaylistLookup map[types.SongKey][]spotify.SimplePlaylist
type TrackGraph struct {
	UserID    string
	Tracks    TrackLookup
	Playlists PlaylistLookup
}

func CreateGraphCache(ctx context.Context) error {
	graph, err := retrieveGraph(ctx)
	if err != nil {
		return err
	}
	return writeGraphToCache(ctx, graph)
}

func GetTrackGraph(ctx context.Context) (TrackGraph, error) {
	graph := TrackGraph{}
	retrieved, err := loadGraphFromCache(ctx, &graph)
	if err != nil || retrieved {
		return graph, err
	}
	return retrieveGraph(ctx)
}

func loadGraphFromCache(ctx context.Context, graph *TrackGraph) (bool, error) {
	contents, err := ioutil.ReadFile(playlistCacheLoc)
	if os.IsNotExist(err) {
		return false, nil
	}
	err = json.Unmarshal(contents, &graph)
	if err != nil {
		log.WithError(err).WithField("file", playlistCacheLoc).Error("Failed to unmarshal cached playlist")
		return false, err
	}
	return true, nil
}

func writeGraphToCache(ctx context.Context, graph TrackGraph) error {
	cache, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(playlistCacheLoc, cache, 0644)
	if err != nil {
		log.WithError(err).WithField("file", playlistCacheLoc).Error("Failed to write playlist cache")
		return err
	}
	return nil
}

func retrieveGraph(ctx context.Context) (TrackGraph, error) {
	graph := TrackGraph{
		Playlists: PlaylistLookup{},
		Tracks:    TrackLookup{},
	}
	client := spotifyclient.ContextClient(ctx)
	opts := &spotify.Options{Limit: &limit}
	playlists, err := client.CurrentUsersPlaylistsOpt(opts)
	if err != nil {
		return graph, fail.Trace(err)
	}
	conf := configuration.ContextConfiguration(ctx)
	log.WithField("regex", conf.Spotify.PlaylistRegex).Debug("Matching playlists")
	matcher, err := regexp.Compile(conf.Spotify.PlaylistRegex)
	if err != nil {
		return graph, fail.Trace(err)
	}
	type result struct {
		UserID   string
		Playlist spotify.SimplePlaylist
		Tracks   []spotify.FullTrack
	}
	ch := make(chan result)
	wg := &sync.WaitGroup{}
	wg2 := &sync.WaitGroup{}
	for _, playlist := range playlists.Playlists {
		wg.Add(1)
		go func(ch chan<- result, wg, wg2 *sync.WaitGroup, playlist spotify.SimplePlaylist) {
			defer wg.Done()
			if !matcher.Match([]byte(playlist.Name)) {
				return
			}
			l := log.WithField("name", playlist.Name)
			l.Info("Processing Playlist")
			var fullList *spotify.PlaylistTrackPage
			for {
				if fullList != nil {
					offset := fullList.Offset + len(fullList.Tracks)
					opts.Offset = &offset
				} else {
					offset := 0
					opts.Offset = &offset
				}
				userID := playlist.Owner.ID
				fullList, err = client.GetPlaylistTracksOpt(playlist.Owner.ID, playlist.ID, opts, "")
				if err != nil {
					log.WithError(err).Fatal()
				}
				if len(fullList.Tracks) == 0 {
					break
				}

				l.WithField("tracks", len(fullList.Tracks)).
					WithField("offset", *opts.Offset).
					Info("Received spotify playlist tracks")
				tracks := []spotify.FullTrack{}
				for _, track := range fullList.Tracks {
					tracks = append(tracks, track.Track)
				}
				wg2.Add(1)
				ch <- result{
					UserID:   userID,
					Playlist: playlist,
					Tracks:   tracks,
				}
			}
		}(ch, wg, wg2, playlist)
	}
	go func(wg2 *sync.WaitGroup) {
		for result := range ch {
			for _, track := range result.Tracks {
				artistParts := []string{}
				for _, part := range track.Artists {
					artistParts = append(artistParts, part.Name)
				}
				key := types.SongKey{
					Artist: strings.Join(artistParts, ", "),
					Title:  track.Name,
				}
				log.WithField("key", key).Debug("Spotify track")
				graph.Tracks[key] = track
				graph.Playlists[key] = append(graph.Playlists[key], result.Playlist)
				graph.UserID = result.UserID
			}
			wg2.Done()
		}
	}(wg2)
	wg.Wait()
	wg2.Wait()
	close(ch)

	log.WithField("total", len(graph.Tracks)).Info("Loaded all spotify playlist tracks")
	return graph, nil
}
