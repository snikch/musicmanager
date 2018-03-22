package music

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const (
	fullCharacter  = "⭑"
	emptyCharacter = "⭒"
)

var (
	mixedInKeyCommentRegex, starRegex *regexp.Regexp
)

func init() {
	r, err := regexp.Compile("^(([0-9]{1,2}[AB])/?([0-9]{1,2}[AB])?|All) - Energy [0-9]{1,2}\\s?-?\\s?")
	if err != nil {
		panic(err)
	}
	mixedInKeyCommentRegex = r
	r, err = regexp.Compile(fmt.Sprintf("([%s%s]+)\\s?-?\\s?", emptyCharacter, fullCharacter))
	if err != nil {
		panic(err)
	}
	starRegex = r
}

// Comment represents a structured comment which contains information about the song
// from various places, such as mixed in key and iTunes.
type Comment struct {
	Key, Energy, Comment string
	Rating               int
}

// ParseComment attempts to parse a raw string into a comment value
func ParseComment(ctx context.Context, raw string) Comment {
	var comment Comment
	// Grab the current rating from the comment.
	stars := string(starRegex.Find([]byte(raw)))
	comment.Rating = strings.Count(stars, fullCharacter)

	// Check for the mixed in key values, camelot key and energy.
	mikValue := string(mixedInKeyCommentRegex.Find([]byte(raw)))
	mikParts := strings.Split(strings.TrimRight(mikValue, " - "), " - ")
	if len(mikParts) == 2 {
		comment.Key = mikParts[0]
		comment.Energy = mikParts[1]
	}

	// Now set the comment to whatever is left when you remove the rating and mik strings.
	comment.Comment = strings.Replace(strings.Replace(raw, mikValue, "", 1), stars, "", 1)
	return comment
}

// String returns the comment string in the format "[MIK Key - MIK Energy][ - Rating][ - Comment]""
func (comment Comment) String() string {
	parts := make([]string, 0, 4)
	if comment.Key != "" && comment.Energy != "" {
		parts = append(parts, comment.Key, comment.Energy)
	}
	if comment.Rating > 0 && comment.Rating <= 5 {
		parts = append(
			parts,
			strings.Repeat(fullCharacter, comment.Rating)+strings.Repeat(emptyCharacter, 5-comment.Rating),
		)
	}
	if comment.Comment != "" {
		parts = append(parts, comment.Comment)
	}
	return strings.Join(parts, " - ")
}

// Filter will remove any of the supplied filters from the comment.
func (comment Comment) Filter(filters []string) {
	for _, filter := range filters {
		comment.Comment = strings.Replace(comment.Comment, filter, "", -1)
	}
}
