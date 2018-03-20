package main

import (
	"context"
	"fmt"
	"os"

	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/commands"
	"github.com/snikch/musicmanager/commands/follow_artists"
	"github.com/snikch/musicmanager/configstore"
	"github.com/snikch/musicmanager/configuration"
	"github.com/snikch/musicmanager/spotifyclient"
)

func main() {
	ctx := configuration.ContextWithConfiguration(context.Background())
	defer func() {
		log.Debug("Saving configuration to file")
		conf := configuration.ContextConfiguration(ctx)
		err := configstore.Save(conf)
		if err != nil {
			log.WithError(err).Error("Could not save configuration")
		}
	}()
	ctx, err := spotifyclient.ContextWithClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.WithField("os.Args", os.Args).Debug("Args")
	if len(os.Args) <= 1 {
		displayHelp()
	}

	switch os.Args[1] {
	case "follow-artists":
		err = follow_artists.Command(ctx)
	case "refresh-spotify":
		err = commands.RefreshSpotify(ctx)
	case "tag-files":
		err = commands.TagFiles(ctx)
	case "create-missing-playlist":
		err = commands.CreateMissingPlaylist(ctx)
	default:
		displayHelp()
	}
	if err != nil {
		log.WithError(err).Fatal()
	}
}

func displayHelp() {
	fmt.Println("Select an arg: refresh-spotify tag-files create-missing-playlist")
	os.Exit(1)
}
