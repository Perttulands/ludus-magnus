package cmd

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func newPrefixedID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, strings.ReplaceAll(uuid.NewString(), "-", "")[:8])
}
