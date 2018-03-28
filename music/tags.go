package music

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/snikch/musicmanager/configuration"
	"github.com/zmb3/spotify"

	"github.com/sirupsen/logrus"
	"github.com/snikch/api/fail"
	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/spotifyclient"
	"github.com/snikch/musicmanager/types"
)

var (
	tagProcessors = map[string]func(context.Context, *logrus.Entry, types.FileWithContext) (bool, error){
		"year":    updateYear,
		"comment": updateComment,
		"genre":   updateGenre,
	}
)

func updateFileWithPlaylistTags(ctx context.Context, fileContext types.FileWithContext) error {
	l := log.WithField("title", fileContext.Title()).
		WithField("artist", fileContext.Artist())
	anyUpdate := false
	for name, processor := range tagProcessors {
		didUpdate, err := processor(ctx, l.WithField("processor", name), fileContext)
		if err != nil {
			return err
		}
		if !anyUpdate && didUpdate {
			anyUpdate = true
		}
	}
	if !anyUpdate {
		return nil
	}
	return fail.Trace(fileContext.Save())
}

func updateYear(ctx context.Context, l *logrus.Entry, fileContext types.FileWithContext) (bool, error) {
	if fileContext.SpotifyTrack == nil {
		return false, nil
	}
	didUpdate := false
	if fileContext.Year() == "" {
		client := spotifyclient.ContextClient(ctx)
		album, err := client.GetAlbum(fileContext.SpotifyTrack.Album.ID)
		if err != nil {
			log.WithError(err).WithField("id", fileContext.SpotifyTrack.Album.ID).Error("Failed to get album from spotify to update Year")
			return false, err
		}
		year := strconv.Itoa(album.ReleaseDateTime().Year())
		fileContext.SetYear(year)
		didUpdate = true
		l.WithField("year", year).Info("Set Year tag")
	}
	return didUpdate, nil
}

// updateComment adds a star rating to the comments of a file and cleans out the crap.
func updateComment(ctx context.Context, l *logrus.Entry, fileContext types.FileWithContext) (bool, error) {
	if fileContext.ITunesTrack == nil {
		l.Debug("No itunes track supplied for rating")
		return false, nil
	}

	oldComment := fileContext.Comment()
	comment := ParseComment(ctx, oldComment)
	comment.Rating = fileContext.ITunesTrack.Rating / 20 // iTunes stores a 1 as 20, 2 -> 40 etc.
	// Remove any shit we don't like in comments.
	comment.Filter(configuration.ContextConfiguration(ctx).MusicFiles.CommentRemovals)
	comment.RemoveGarbage()
	newComment := comment.String()
	if oldComment == newComment {
		l.WithField("old", oldComment).
			WithField("itunes", fileContext.ITunesTrack).
			Debug("No change in comment")
		return false, nil
	}
	l.WithField("old", oldComment).
		WithField("new", newComment).
		Info("Updating Comment")
	fileContext.SetComment(newComment)
	return true, nil
}

func updateGenre(ctx context.Context, l *logrus.Entry, fileContext types.FileWithContext) (bool, error) {
	removals := removalTags(ctx)
	playlist, removedPlaylist := removeTags(playlistTags(ctx, fileContext.SpotifyPlaylists), removals)
	current, removedCurrent := removeTags(currentTags(replaceTags(ctx, fileContext.Genre())), removals)
	target := mergeTags(playlist, current)
	tags := flattenTags(target)
	sort.Strings(tags)
	l = l.WithField("playlist", flattenTags(playlist)).
		WithField("current", fileContext.Genre()).
		WithField("removed", flattenTags(mergeTags(removedPlaylist, removedCurrent))).
		WithField("target", tags)
	genre := strings.Join(tags, " ")
	if genre == fileContext.Genre() {
		l.Debug("No genre update required")
		return false, nil
	}
	added, _ := removeTags(target, current)
	l.WithField("added", flattenTags(added)).
		Info("Adjusting tags")
	fileContext.SetGenre(genre)
	return true, nil
}

func playlistNameToGenres(name string) []string {
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

func playlistTags(ctx context.Context, playlists []spotify.SimplePlaylist) map[string]bool {
	lookup := map[string]bool{}
	for _, playlist := range playlists {
		name := replaceTags(ctx, playlist.Name)
		for _, genre := range playlistNameToGenres(name) {
			lookup[strings.ToLower(genre)] = true
		}
	}
	return lookup
}
func currentTags(genre string) map[string]bool {
	lookup := map[string]bool{}
	for _, genre := range strings.Split(genre, " ") {
		if genre == "" {
			continue
		}
		lookup[strings.ToLower(genre)] = true
	}
	return lookup
}
func removeTags(tags, removals map[string]bool) (map[string]bool, map[string]bool) {
	removed := map[string]bool{}
	for tag := range tags {
		if !removals[tag] {
			continue
		}
		removed[tag] = true
		delete(tags, tag)
	}
	return tags, removed
}

func flattenTags(lookup map[string]bool) []string {
	tags := make([]string, len(lookup))
	i := 0
	for tag := range lookup {
		tags[i] = tag
		i++
	}
	return tags
}

func replaceTags(ctx context.Context, genre string) string {
	conf := configuration.ContextConfiguration(ctx)
	for search, replace := range conf.MusicFiles.TagReplacements {
		genre = strings.Replace(genre, search, replace, -1)
	}
	return genre
}

func removalTags(ctx context.Context) map[string]bool {
	removals := map[string]bool{}
	conf := configuration.ContextConfiguration(ctx)
	for _, tag := range conf.MusicFiles.TagRemovals {
		removals[strings.ToLower(tag)] = true
	}
	return removals
}

func mergeTags(a, b map[string]bool) map[string]bool {
	for tag := range a {
		b[tag] = true
	}
	return b
}
