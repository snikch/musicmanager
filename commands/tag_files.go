package commands

import (
	"context"

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
	return music.UpdateFilesWithTagsFromGraph(ctx, files, graph)
}
