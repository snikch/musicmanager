package spotifyclient

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/skratchdot/open-golang/open"
	"github.com/snikch/api/config"
	"github.com/snikch/api/log"
	"github.com/snikch/api/vc"
	"github.com/snikch/musicmanager/configuration"
	"github.com/zmb3/spotify"
)

var (
	// stateToken is a unique nonce to ensure request hijackin of the oauth flow doesn't occur.
	stateToken = generateStateToken()
	// authToken is the persisted authToken
	authToken string
	// authenticator is the spotify authentication service
	authenticator spotify.Authenticator
	client        *spotify.Client
	port          = config.String("SPOTIFY_AUTHENTICATION_PORT", "4000")
	redirectURL   = "http://127.0.0.1:" + port
)

func init() {
	scopes := []string{
		spotify.ScopeUserReadPrivate,
		spotify.ScopePlaylistReadPrivate,
		spotify.ScopePlaylistModifyPrivate,
		spotify.ScopePlaylistModifyPublic,
		spotify.ScopeUserFollowRead,
		spotify.ScopeUserFollowModify,
	}
	authenticator = spotify.NewAuthenticator(redirectURL, scopes...)
	authenticator.SetAuthInfo(config.String("SPOTIFY_CLIENT_ID", ""), config.PrivateString("SPOTIFY_CLIENT_SECRET"))
}

type contextKey int

const clientKey contextKey = iota

// ContextWithClient returns an authenticated client for spotify access.
func ContextWithClient(ctx context.Context) (context.Context, error) {
	conf := configuration.ContextConfiguration(ctx)
	if conf.Spotify.AuthToken != nil {
		log.Debug("Creating client from existing auth token")

		client := authenticator.NewClient(conf.Spotify.AuthToken)
		client.AutoRetry = true
		return context.WithValue(ctx, clientKey, &client), nil
	}
	err := authenticateUser(ctx)
	return context.WithValue(ctx, clientKey, client), err
}

func ContextClient(ctx context.Context) *spotify.Client {
	val := ctx.Value(clientKey)
	if val != nil {
		return val.(*spotify.Client)
	}
	return nil
}

// authenticateUser starts the oauth authentication flow.
func authenticateUser(ctx context.Context) error {
	shutdown := make(chan bool)
	server := &http.Server{
		Addr: ":" + port,
		// func handles the oauth flow response and token retrieval.
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// use the same state string here that you used to generate the URL
			token, err := authenticator.Token(stateToken, r)
			if err != nil {
				vc.RespondWithError(w, r, err)
				return
			}
			conf := configuration.ContextConfiguration(ctx)
			conf.Spotify.AuthToken = token
			// Create a client using the specified token.
			c := authenticator.NewClient(token)
			client = &c
			// Start the shutdown process
			shutdown <- true
		}),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.WithError(err).Error("Server stopped")
		}
	}()
	open.Run(authenticator.AuthURL(stateToken))
	<-shutdown
	err := server.Close()
	if err != nil {
		log.WithError(err).Error(`Server failed to close ¯\_(ツ)_/¯`)
	}
	return nil
}

func generateStateToken() string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%x", rnd.Uint64())
}
