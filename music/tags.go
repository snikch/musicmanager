package music

import (
	"context"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/snikch/musicmanager/configuration"

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
	targetTags := map[string]bool{}
	tagRemovals := map[string]bool{}
	conf := configuration.ContextConfiguration(ctx)
	removalLookup := map[string]bool{}
	for _, tag := range conf.MusicFiles.TagRemovals {
		removalLookup[tag] = true
	}
	for _, playlist := range fileContext.SpotifyPlaylists {
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
	for _, genre := range strings.Split(fileContext.Genre(), " ") {
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
		return false, nil
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
		return false, nil
	}
	l.WithField("added", addedTags).
		WithField("all", genres).
		Info("Adjusting tags")
	sort.Strings(genres)
	fileContext.SetGenre(strings.Join(genres, " "))
	return true, nil
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
