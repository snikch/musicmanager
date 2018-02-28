package commands

import (
	"context"

	"github.com/snikch/musicmanager/spotify"
)

// RefreshSpotify retrieves all playlists from spotify and stores them in a cache.
func RefreshSpotify(ctx context.Context) error {
	return spotify.CreateGraphCache(ctx)
}
