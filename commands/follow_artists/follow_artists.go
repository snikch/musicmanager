package follow_artists

import (
	"context"
	"strings"

	"github.com/snikch/api/fail"
	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/itunes"
	"github.com/snikch/musicmanager/spotifyclient"
	"github.com/zmb3/spotify"
)

const (
	concurrency = 2
)

// Command ensures all artists with a 3 star rating or higher are followed on spotify.
func Command(ctx context.Context) error {
	loc := configuration.ContextConfiguration(ctx).ITunes.Dir + "iTunes Music Library.xml"
	library, err := itunes.LoadLibrary(loc)
	if err != nil {
		return err
	}
	tracks := itunes.FilterRating(library.Tracks, 3)
	artists := itunes.ReduceArtists(tracks)
	log.
		WithField("tracks", len(tracks)).
		WithField("artists", len(artists)).
		Info("Found tracks with rating 3+")
	manager := newFollowManager(ctx)
	return manager.ensureFollowed(ctx, artists)
}

type followManager struct {
	client          *spotify.Client
	conf            *configuration.Configuration
	skipLookup      map[string]bool
	followingLookup map[string]bool
}

func newFollowManager(ctx context.Context) *followManager {
	manager := &followManager{
		client:     spotifyclient.ContextClient(ctx),
		conf:       configuration.ContextConfiguration(ctx),
		skipLookup: map[string]bool{},
	}
	conf := configuration.ContextConfiguration(ctx)
	for _, skip := range conf.ITunes.Artists.Skip {
		manager.skipLookup[skip] = true
	}
	return manager
}

func (manager *followManager) ensureFollowed(ctx context.Context, names []string) error {
	following, err := getFollowing(ctx)
	if err != nil {
		return err
	}
	manager.followingLookup = followingLookup(following)
	names = splitNames(names)
	toFollow := []spotify.ID{}
	for _, name := range names {
		id, err := manager.shouldFollow(ctx, name)
		if err != nil {
			log.WithField("iTunes artist", name).WithError(err).Error("Failed to determine follow state")
			continue
		}
		if id != nil {
			toFollow = append(toFollow, *id)
		}
	}
	if len(toFollow) == 0 {
		log.Info("No artists to follow")
		return nil
	}
	log.WithField("count", len(toFollow)).Info("Artists to follow")
	client := spotifyclient.ContextClient(ctx)
	cursor := 0
	for {
		head := cursor
		tail := head + 50
		if tail > len(toFollow) {
			tail = len(toFollow)
		}
		log.WithField("count", tail-head).Info("Following artists")
		// We can only follow in lots of 50
		err := client.FollowArtist(toFollow[head:tail]...)
		if err != nil {
			return fail.Trace(err)
		}
		// If we've hit the end, break out
		if tail-head < 50 || tail == len(toFollow) {
			break
		}
		cursor = tail
	}
	return nil
}

func (manager *followManager) shouldFollow(ctx context.Context, name string) (*spotify.ID, error) {
	if name == "" {
		return nil, nil
	}
	l := log.
		WithField("iTunes artist", name)
	if manager.skipLookup[name] {
		l.Debug("Skipping at request of config ITunes.Artists.Skip")
		return nil, nil
	}
	if override, ok := manager.conf.ITunes.Artists.SpotifyOverrides[name]; ok {
		l.
			WithField("override", override).
			Debug("Overriding iTunes name with config ITunes.Artists.SpotifyOverrides value")
		name = override
	}
	name = strings.ToLower(name)
	if manager.followingLookup[name] {
		l.Debug("Already following")
		return nil, nil
	}
	result, err := manager.client.Search(name, spotify.SearchTypeArtist)
	if err != nil {
		return nil, fail.Trace(err)
	}
	if len(result.Artists.Artists) == 0 {
		l.Warn("Could not find Spotify artist")
		return nil, nil
	}
	spotifyArtist := result.Artists.Artists[0]
	l = l.WithField("Spotify artist", spotifyArtist.Name)
	if strings.ToLower(spotifyArtist.Name) != name {
		l.
			WithField("Spotify URI", spotifyArtist.URI).
			Warn("Name mismatch. Add an artist override in config.json ITunes.Artists.SpotifyOverrides to force a match")
		return nil, nil
	}
	l.Info("Will follow")
	return &spotifyArtist.ID, nil
}

func splitNames(names []string) []string {
	out := []string{}
	for _, name := range names {
		out = append(out, strings.Split(name, ", ")...)
	}
	return out
}

func getFollowing(ctx context.Context) ([]spotify.FullArtist, error) {
	client := spotifyclient.ContextClient(ctx)
	limit := 50
	artists := []spotify.FullArtist{}
	after := ""
	for {
		chunk, err := client.CurrentUsersFollowedArtistsOpt(limit, after)
		if err != nil {
			return nil, fail.Trace(err)
		}
		artists = append(artists, chunk.Artists...)
		if len(chunk.Artists) < limit {
			break
		}
		after = chunk.Cursor.After
	}
	return artists, nil
}

func followingLookup(artists []spotify.FullArtist) map[string]bool {
	lookup := map[string]bool{}
	for _, artist := range artists {
		lookup[strings.ToLower(artist.Name)] = true
	}
	return lookup
}
