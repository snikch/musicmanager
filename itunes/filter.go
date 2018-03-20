package itunes

// FilterRating returns only the tracks with a rating of min or more.
func FilterRating(tracks map[string]Track, min int) []Track {
	out := []Track{}
	for _, track := range tracks {
		if track.Rating < min {
			continue
		}
		out = append(out, track)
	}
	return out
}

// ReduceArtists returns a slice of artist names from the supplied tracks.
func ReduceArtists(tracks []Track) []string {
	lookup := map[string]bool{}
	out := []string{}
	for _, track := range tracks {
		if lookup[track.Artist] {
			continue
		}
		out = append(out, track.Artist)
		lookup[track.Artist] = true
	}
	return out
}
