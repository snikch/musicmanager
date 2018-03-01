package configuration

import (
	"context"

	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configstore"
	"golang.org/x/oauth2"
)

type Configuration struct {
	Spotify struct {
		AuthToken      *oauth2.Token
		PlaylistRegex  string
		OutputPlaylist struct {
			ID   string
			Name string
		}
	}
	MusicFiles struct {
		TagReplacements map[string]string
		TagRemovals     []string
		Dirs            []string
	}
}

type contextKey int

const configKey contextKey = iota

// ContextConfiguration returns the configuration for the supplied context.
func ContextConfiguration(ctx context.Context) *Configuration {
	val := ctx.Value(configKey)
	if val != nil {
		return val.(*Configuration)
	}
	return nil
}

// ContextWithConfiguration returns a new context with a configuration loaded.
func ContextWithConfiguration(ctx context.Context) context.Context {
	configuration := &Configuration{}
	err := configstore.Load(configuration)
	if err != nil {
		log.WithError(err).Fatal("Could not load configuration file")
	}
	return context.WithValue(ctx, configKey, configuration)

}
