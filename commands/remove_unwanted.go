package commands

import (
	"context"

	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/itunes"
	"github.com/snikch/musicmanager/music"
	"github.com/snikch/musicmanager/spotify"
)

// RemoveUnwanted removes all files with the delete tag from iTunes and any Spotify playlists.
func RemoveUnwanted(ctx context.Context) error {
	graph, err := spotify.GetTrackGraph(ctx)
	if err != nil {
		return err
	}
	files, err := music.GetAllFiles(ctx)
	if err != nil {
		return err
	}
	loc := configuration.ContextConfiguration(ctx).ITunes.Dir + "iTunes Music Library.xml"
	library, err := itunes.LoadLibrary(loc)
	if err != nil {
		return err
	}
	contexts := music.FilesToFileContexts(ctx, files)
	contexts = music.HydrateSpotifyOnContexts(ctx, contexts, graph)
	contexts = music.HydrateITunesOnContexts(ctx, contexts, library)

	didRemove, err := music.RemoveUnwanted(ctx, graph, contexts)
	if err != nil {
		return err
	}
	if !didRemove {
		return nil
	}
	log.Info("Refreshing Spotify cache after removing tracks")
	return spotify.CreateGraphCache(ctx)
}
