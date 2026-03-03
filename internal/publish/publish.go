package publish

import (
	"errors"
	"fmt"
	"strings"
)

func ValidateText(text string, maxWords int) error {
	if strings.TrimSpace(text) == "" {
		return errors.New("text cannot be empty")
	}

	words := strings.Fields(text)
	if len(words) > maxWords {
		return fmt.Errorf("max %d words allowed", maxWords)
	}

	return nil
}
