package music

import (
	"context"
	"testing"

	"github.com/snikch/api/log"
	"github.com/snikch/musicmanager/configuration"
)

func TestParse(t *testing.T) {
	ctx := configuration.ContextWithConfiguration(context.Background())
	for i, test := range []struct {
		Raw     string
		Comment Comment
	}{
		{"", testComment("", "", "", 0)},
		{"Testing just a comment", testComment("", "", "Testing just a comment", 0)},
		{"xxxx+", testComment("", "", "", 4)},
		{"4A - Energy 2 - Basic MIK", testComment("4A", "Energy 2", "Basic MIK", 0)},
		{"4A/5A - Energy 2 - Mixed MIK", testComment("4A/5A", "Energy 2", "Mixed MIK", 0)},
		{"All - Energy 2 - All MIK", testComment("All", "Energy 2", "All MIK", 0)},
		{"6A - Energy 2 - xxx++ - MIK + Rating", testComment("6A", "Energy 2", "MIK + Rating", 3)},
		{"7A - Energy 2 - xx+++ - Everything", testComment("7A", "Energy 2", "Everything", 2)},
	} {
		comment := ParseComment(ctx, test.Raw)
		if comment.Key != test.Comment.Key {
			t.Fatalf("Key: Expected %s to match %s", comment.Key, test.Comment.Key)
		}
		if comment.Energy != test.Comment.Energy {
			t.Fatalf("Energy: Expected %s to match %s", comment.Energy, test.Comment.Energy)
		}
		if comment.Rating != test.Comment.Rating {
			t.Fatalf("Rating %d: Expected %d to match %d", i, comment.Rating, test.Comment.Rating)
		}
		if comment.Comment != test.Comment.Comment {
			t.Fatalf("Comment: Expected %s to match %s", comment.Comment, test.Comment.Comment)
		}
	}
}

func TestGarbage(t *testing.T) {
	comment := Comment{
		Comment: "Testing 0000041A 00000329 000024F4 0000193F 00023068 0001CA5E 00004A5B 00004AFE 0002AD4E 000480DB 00000000 00000210 00000AC5 0000000001398B2B 00000000 011C4FAD 00000000 00000000 00000000 00000000 00000000 00000000",
	}
	comment.RemoveGarbage()
	if comment.Comment != "Testing" {
		t.Fatalf("Expected just 'Testing', got %s", comment.Comment)
	}
	log.WithField("out", comment.Comment).Debug("Comment")
}

func testComment(key, energy, comment string, rating int) Comment {
	return Comment{
		Key:     key,
		Energy:  energy,
		Comment: comment,
		Rating:  rating,
	}
}
