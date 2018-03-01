# Music Manager

Manage and tag your local music files from Spotify Playlists. I find music on Spotify, put them into playlists like `House: Vocal` and `House: Funky`. I use the `create-missing-playlist` command to create a `Missing` playlist that I then download via whatever means (Beatport, Ripping, whatever). Once downloaded, I use the `tag-files` command to update my local music Mp3's with tags based on the playlists, so I end up with a Genre ID3 tag such as `house vocal funky`. From there I can make smart playlists in iTunes based on those tags, and use that in whatever software I want (Djay Pro, Serato etc.).

## Install

```sh
go install github.com/snikch/musicmanager
```

Copy `.env.example` to `.env` and enter your Spotify application credentials.

Copy `config.example.json` to `.config.json` and update for your configuration.

## Commands

e.g.

```
musicmanager refresh-spotify
```

### refresh-spotify

Pulls down matching playlists from Spotify and stores them in a cache.

### create-missing-playlists

Compares local files against Spotify playlists and creates a new Spotify playlist with all tracks that aren't present locally. You can now purchase these from Beatport or get them from whereever.

### tag-files

Tag local files with the following:

* Genre: Converts playlist names to tags (e.g. House: Vocal becomes "house" and "vocal"), and adds them in the Genre ID3 tag
* Year: Ensures the Year ID3 tag is set
