package music

import (
	"context"
	"testing"

	"github.com/snikch/musicmanager/configuration"
)

func TestParse(t *testing.T) {
	ctx := configuration.ContextWithConfiguration(context.Background())
	for _, test := range []struct {
		Raw     string
		Comment Comment
	}{
		{"", testComment("", "", "", 0)},
		{"Testing just a comment", testComment("", "", "Testing just a comment", 0)},
		{"⭑⭑⭑⭑⭒", testComment("", "", "", 4)},
		{"4A - Energy 2 - Basic MIK", testComment("4A", "Energy 2", "Basic MIK", 0)},
		{"4A/5A - Energy 2 - Mixed MIK", testComment("4A/5A", "Energy 2", "Mixed MIK", 0)},
		{"All - Energy 2 - All MIK", testComment("All", "Energy 2", "All MIK", 0)},
		{"6A - Energy 2 - ⭑⭑⭑⭒⭒ - MIK + Rating", testComment("6A", "Energy 2", "MIK + Rating", 3)},
		{"7A - Energy 2 - ⭑⭑⭒⭒⭒ - Everything", testComment("7A", "Energy 2", "Everything", 2)},
	} {
		comment := ParseComment(ctx, test.Raw)
		if comment.Key != test.Comment.Key {
			t.Fatalf("Key: Expected %s to match %s", comment.Key, test.Comment.Key)
		}
		if comment.Energy != test.Comment.Energy {
			t.Fatalf("Energy: Expected %s to match %s", comment.Energy, test.Comment.Energy)
		}
		if comment.Rating != test.Comment.Rating {
			t.Fatalf("Rating: Expected %d to match %d", comment.Rating, test.Comment.Rating)
		}
		if comment.Comment != test.Comment.Comment {
			t.Fatalf("Comment: Expected %s to match %s", comment.Comment, test.Comment.Comment)
		}
	}
}

func testComment(key, energy, comment string, rating int) Comment {
	return Comment{
		Key:     key,
		Energy:  energy,
		Comment: comment,
		Rating:  rating,
	}
}
