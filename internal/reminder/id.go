package reminder

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// GenerateID returns a new ULID string, sortable and URL-safe.
func GenerateID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
