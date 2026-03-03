package random

import (
	"math/rand"
	"time"
)

var fallbackLibrary = []string{
	"Someone out there believes in your next step.",
	"Small progress still moves your story forward.",
	"A quiet day can still hold something meaningful.",
	"You are allowed to begin again today.",
	"Kindness to yourself counts as momentum.",
}

func FallbackSentence() string {
	if len(fallbackLibrary) == 0 {
		return "No sentence available right now."
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fallbackLibrary[rng.Intn(len(fallbackLibrary))]
}
