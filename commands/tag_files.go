package commands

import (
	"context"

	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/itunes"

	"github.com/snikch/musicmanager/music"
	"github.com/snikch/musicmanager/spotify"
)

func TagFiles(ctx context.Context) error {
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
	return music.UpdateFilesWithTagsFromGraphAndLibrary(ctx, files, graph, library)
}
