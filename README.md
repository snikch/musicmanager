# Music Manager

Manage and tag your local music files from Spotify Playlists. I find music on Spotify, put them into playlists like
`House: Vocal` and `House: Funky`. I use the `create-missing-playlist` command to create a `Missing` playlist that I
then download via whatever means (Beatport, Ripping, whatever). Once downloaded, I use the `tag-files` command to update
my local music Mp3's with tags based on the playlists, so I end up with a Genre ID3 tag such as `house vocal funky`.
From there I can make smart playlists in iTunes based on those tags, and use that in whatever software I want (Djay Pro,
Serato etc.).

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

Compares local files against Spotify playlists and creates a new Spotify playlist with all tracks that aren't present
locally. You can now purchase these from Beatport or get them from whereever.

### tag-files

Tag local files with the following:

* Genre: Converts playlist names to tags (e.g. House: Vocal becomes "house" and "vocal"), and adds them in the Genre ID3
  tag
* Year: Ensures the Year ID3 tag is set (from album information in Spotify)
* Comment: Adds the iTunes rating to the comments (if one exists)

### follow-artists

Follows on Spotify all artists with a 3⭐ rating or higher.

## Future Commands

To be written

* [ ] `create-following-playlist` Creates a playlist of all songs from followed Spotify artists that were released after
      a given date (persisted after running to allow continuous running of the command)
* [ ] `find-longer` Find longer versions of songs in Spotify (such as the extended mix).
* [ ] `find-longer` Add beatport support since Spotify often only has radio edits
* [ ] `remove-unwanted` Remove local files with a specific tag from all Spotify playlists, delete local file and remove
      from iTunes
* [ ] `backpropagate-tags` Push tags from local files back to Spotify playlists
