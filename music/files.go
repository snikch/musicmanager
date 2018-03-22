package music

import (
	"context"
	"io/ioutil"
	"path"

	"github.com/bogem/id3v2"
	id3 "github.com/mikkyang/id3-go"
	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/itunes"
	"github.com/snikch/musicmanager/spotify"
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
		log.WithField("files", len(dirFiles)).WithField("dir", dir).Debug("Found music files in dir")
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
			Info("Found Music Track")
		files = append(files, f)
	}
	return files, nil
}

func UnionFilesWithGraph(ctx context.Context, files []types.File, graph spotify.TrackGraph) []types.FileWithSpotifyTrack {
	union := make([]types.FileWithSpotifyTrack, 0, len(files))
	for _, file := range files {
		key := fileKey(file)
		if key.Artist == "" || key.Title == "" {
			log.WithField("key", key).
				WithField("name", file.Filename).
				Debug("Skipping unknown track")
			continue
		}
		if track, exists := graph.Tracks[key]; exists {
			log.WithField("key", key).WithField("spotify id", track.ID.String()).Debug("Found track")
			union = append(union, types.FileWithSpotifyTrack{
				File:         file,
				SpotifyTrack: track,
			})
		} else {
			log.WithField("key", key).Debug("Couldn't find spotify track")
		}
	}
	return union
}

func UpdateFilesWithTagsFromGraphAndLibrary(ctx context.Context, files []types.File, graph spotify.TrackGraph, library *itunes.Library) error {
	trackLookup := map[types.SongKey]*itunes.Track{}
	for _, track := range library.Tracks {
		trackLookup[types.SongKey{
			Title:  track.Name,
			Artist: track.Artist,
		}] = &track
	}
	for _, tuple := range UnionFilesWithGraph(ctx, files, graph) {
		track := trackLookup[types.SongKey{
			Title:  tuple.File.Title(),
			Artist: tuple.File.Artist(),
		}]
		err := updateFileWithPlaylistTags(ctx, tuple.File, tuple.SpotifyTrack, graph.Playlists[fileKey(tuple.File)], track)
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
