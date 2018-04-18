package music

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/bogem/id3v2"
	"github.com/everdev/mack"
	id3 "github.com/mikkyang/id3-go"
	"github.com/snikch/api/fail"
	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/itunes"
	"github.com/snikch/musicmanager/spotify"
	"github.com/snikch/musicmanager/spotifyclient"
	"github.com/snikch/musicmanager/types"
)

func GetAllFiles(ctx context.Context) ([]types.File, error) {
	files := []types.File{}
	conf := configuration.ContextConfiguration(ctx)
	for _, dir := range conf.MusicFiles.Dirs {
		log.WithField("dir", dir).Info("Loading music files from dir")
		dirFiles, err := loadDir(dir)
		if err != nil {
			return nil, err
		}
		files = append(files, dirFiles...)
	}
	log.WithField("total", len(files)).Info("Found local music files")
	return files, nil
}

func loadDir(loc string) ([]types.File, error) {
	dir, err := ioutil.ReadDir(loc)
	if err != nil {
		return nil, err
	}
	files := make([]types.File, 0, len(dir))
	log.WithField("dir", loc).WithField("files", len(dir)).Debug("Found files")
	for _, dirFile := range dir {
		if dirFile.IsDir() {
			subFiles, err := loadDir(loc + "/" + dirFile.Name())
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
			continue
		}
		ext := path.Ext(dirFile.Name())
		if ext != ".mp3" && ext != ".m4a" {
			log.WithField("name", dirFile.Name()).Debug("Skipping invalid extension")
			continue
		}
		var song types.Song
		v2Song, err := id3v2.Open(loc+"/"+dirFile.Name(), id3v2.Options{Parse: true})
		if err != nil {
			file, err := id3.Open(loc + "/" + dirFile.Name())
			if err != nil {
				return nil, err
			}
			comments := file.Comments()
			if len(comments) > 0 {
				log.WithField("comments", comments[0]).
					WithField("len", len(comments)).
					Warn("Comments")
			}
			song = types.ID3Wrapper{file}
		} else {
			song = types.ID3V2Wrapper{v2Song}
		}
		f := types.File{
			Song:     types.CleanWrapper{song},
			Filename: dirFile.Name(),
			Dir:      loc,
		}

		log.WithField("name", dirFile.Name()).
			WithField("title", song.Title()).
			WithField("comment", f.Comment()).
			Debug("Found Music Track")
		files = append(files, f)
	}
	return files, nil
}

func FilesToFileContexts(ctx context.Context, files []types.File) types.FileContexts {
	out := types.FileContexts{}
	for i := range files {
		out[fileKey(files[i])] = types.FileWithContext{File: files[i]}
	}
	return out
}

func HydrateSpotifyOnContexts(ctx context.Context, contexts types.FileContexts, graph spotify.TrackGraph) types.FileContexts {
	for key, fileContext := range contexts {
		if track, exists := graph.Tracks[key]; exists {
			log.WithField("key", key).WithField("spotify id", track.ID.String()).Debug("Found track")
			fileContext.SpotifyTrack = &track
			fileContext.SpotifyPlaylists = graph.Playlists[key]
			contexts[key] = fileContext
		} else {
			log.WithField("key", key).Debug("Couldn't find spotify track")
		}
	}
	return contexts
}

func HydrateITunesOnContexts(ctx context.Context, contexts types.FileContexts, library *itunes.Library) types.FileContexts {
	trackLookup := map[types.SongKey]itunes.Track{}
	for key := range library.Tracks {
		track := library.Tracks[key]
		trackLookup[types.SongKey{
			Title:  track.Name,
			Artist: track.Artist,
		}] = track
	}
	for key, fileContext := range contexts {
		track, ok := trackLookup[key]
		if ok {
			fileContext.ITunesTrack = &track
		}
		contexts[key] = fileContext
	}
	return contexts
}

func UpdateFilesTags(ctx context.Context, contexts types.FileContexts) error {
	for _, fileContext := range contexts {
		err := updateFileWithPlaylistTags(ctx, fileContext)
		if err != nil {
			return err
		}
	}
	return nil
}

func fileKey(file types.File) types.SongKey {
	return types.SongKey{
		Artist: file.Artist(),
		Title:  file.Title(),
	}
}

func LeftOuterJoinFilesToGraph(ctx context.Context, files []types.File, graph spotify.TrackGraph) spotify.TrackLookup {
	tracks := graph.Tracks
	for _, file := range files {
		key := fileKey(file)
		if key.Artist == "" || key.Title == "" {
			log.WithField("key", key).
				WithField("name", file.Filename).
				Debug("Skipping unknown track")
			continue
		}
		if track, exists := tracks[key]; exists {
			log.WithField("key", key).WithField("spotify id", track.ID.String()).Debug("Found track")
			delete(tracks, key)
		} else {
			log.WithField("key", key).Debug("Couldn't find spotify track for local file")
		}
	}
	return tracks
}

func RemoveUnwanted(ctx context.Context, graph spotify.TrackGraph, contexts types.FileContexts) (bool, error) {
	didRemove := false
	client := spotifyclient.ContextClient(ctx)
	deleteTag := configuration.ContextConfiguration(ctx).MusicFiles.DeleteTag
	if deleteTag == "" {
		deleteTag = "delete"
	}
	log.WithField("tag", deleteTag).Info("Starting to remove unwanted tracks")
	for _, fileContext := range contexts {
		key := fileKey(fileContext.File)
		l := log.WithField("key", key)

		if !strings.Contains(fileContext.Genre(), deleteTag) {
			continue
		}

		l.Warn("Will remove")

		if fileContext.SpotifyTrack != nil {
			for _, playlist := range fileContext.SpotifyPlaylists {
				didRemove = true
				l.WithField("playlist", playlist.Name).
					WithField("trackID", fileContext.SpotifyTrack.ID).
					WithField("playlistID", playlist.ID).
					Warn("Removing track from playlist")
				_, err := client.RemoveTracksFromPlaylist(graph.UserID, playlist.ID, fileContext.SpotifyTrack.ID)
				if err != nil {
					return false, fail.Trace(err)
				}
			}
		}

		if fileContext.ITunesTrack != nil {
			didRemove = true
			l.Warn("Removing track from iTunes")
			// set dbid to database ID of theTrack
			// set theTrack to contents of t
			_, err := mack.Tell("iTunes", fmt.Sprintf(`
				set theTrack to (some track of playlist "Library" whose database ID is %d)
				set floc to (get location of theTrack)
				delete theTrack
				tell application "Finder" to delete floc
				`, fileContext.ITunesTrack.TrackID))
			if err != nil {
				return false, fail.Trace(err)
			}
		}
	}
	return didRemove, nil
}
