package files

import (
	"io/ioutil"
	"path"

	id3 "github.com/mikkyang/id3-go"
	"github.com/snikch/api/log"
)

type File struct {
	*id3.File
	Filename string
	Dir      string
}

func LoadDir(loc string) ([]*File, error) {
	dir, err := ioutil.ReadDir(loc)
	if err != nil {
		return nil, err
	}
	files := make([]*File, 0, len(dir))
	log.WithField("dir", loc).WithField("files", len(dir)).Debug("Found files")
	for _, dirFile := range dir {
		if dirFile.IsDir() {
			subFiles, err := LoadDir(loc + "/" + dirFile.Name())
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
		mp3File, err := id3.Open(loc + "/" + dirFile.Name())
		if err != nil {
			return nil, err
		}
		// log.WithField("name", dirFile.Name()).Debug("Found Music Track")
		files = append(files, &File{
			File:     mp3File,
			Filename: dirFile.Name(),
			Dir:      loc,
		})
	}
	return files, nil
}
