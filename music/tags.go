package music

import (
	"context"
	"reflect"
	"strconv"
	"strings"

	"github.com/snikch/musicmanager/configuration"

	"github.com/sirupsen/logrus"
	"github.com/snikch/api/fail"
	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/spotifyclient"
	"github.com/snikch/musicmanager/types"
	"github.com/zmb3/spotify"
)

func updateFileWithPlaylistTags(ctx context.Context, file types.File, track spotify.FullTrack, playlists []spotify.SimplePlaylist) error {
	l := log.WithField("title", file.Title()).
		WithField("artist", file.Artist())
	didUpdateBasic, err := updateBasics(ctx, l, file, track, playlists)
	if err != nil {
		return err
	}
	didUpdateGenre := updateGenre(ctx, l, file, playlists)
	if !didUpdateBasic && !didUpdateGenre {
		return nil
	}
	return fail.Trace(file.Save())
}

func updateBasics(ctx context.Context, l *logrus.Entry, file types.File, track spotify.FullTrack, playlists []spotify.SimplePlaylist) (bool, error) {
	didUpdate := false
	if file.Year() == "" {
		client := spotifyclient.ContextClient(ctx)
		album, err := client.GetAlbum(track.Album.ID)
		if err != nil {
			log.WithError(err).WithField("id", track.Album.ID).Error("Failed to get album from spotify to update Year")
			return false, err
		}
		year := strconv.Itoa(album.ReleaseDateTime().Year())
		file.SetYear(year)
		didUpdate = true
		l.WithField("year", year).Info("Set Year tag")
	}
	return didUpdate, nil
}

func updateGenre(ctx context.Context, l *logrus.Entry, file types.File, playlists []spotify.SimplePlaylist) bool {
	targetTags := map[string]bool{}
	tagRemovals := map[string]bool{}
	conf := configuration.ContextConfiguration(ctx)
	removalLookup := map[string]bool{}
	for _, tag := range conf.MusicFiles.TagRemovals {
		removalLookup[tag] = true
	}
	for _, playlist := range playlists {
		name := playlist.Name
		for match, replacement := range conf.MusicFiles.TagReplacements {
			name = strings.Replace(name, match, replacement, -1)
		}
		for _, genre := range playlistNameToGenres(ctx, name) {
			if _, ok := removalLookup[genre]; !ok {
				targetTags[genre] = true
			}
		}
	}
	existingTags := map[string]bool{}
	for _, genre := range strings.Split(file.Genre(), " ") {
		if genre == "" {
			continue
		}
		if _, ok := removalLookup[genre]; ok {
			tagRemovals[genre] = true
		} else {
			existingTags[genre] = true
		}
	}
	removingTags := make([]string, 0, len(tagRemovals))
	for tag := range tagRemovals {
		removingTags = append(removingTags, tag)
	}
	l = l.WithField("removals", removingTags).
		WithField("existing", existingTags).
		WithField("target", targetTags)
	if reflect.DeepEqual(targetTags, existingTags) && len(removingTags) == 0 {
		l.Debug("No tag update required")
		return false
	}
	genres := []string{}
	for genre := range existingTags {
		genres = append(genres, genre)
	}
	addedTags := []string{}
	for genre := range targetTags {
		if _, exists := existingTags[genre]; !exists {
			addedTags = append(addedTags, genre)
			genres = append(genres, genre)
		}
	}
	if len(addedTags) == 0 && len(removingTags) == 0 {
		l.Debug("No tags to change")
		return false
	}
	l.WithField("added", addedTags).
		WithField("all", genres).
		Info("Adjusting tags")
	file.SetGenre(strings.Join(genres, " "))
	return true
}

func playlistNameToGenres(ctx context.Context, name string) []string {
	parts := strings.Split(name, " ")
	genres := make([]string, 0, len(parts))
	for i := range parts {
		part := strings.ToLower(strings.Trim(parts[i], " "))
		if part != "" {
			genres = append(genres, part)
		}
	}
	return genres
}
