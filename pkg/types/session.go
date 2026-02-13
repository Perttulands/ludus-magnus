package types

import "time"

const (
	ModeQuickstart = "quickstart"
)

type Session struct {
	ID        string
	Need      string
	Mode      string
	CreatedAt time.Time
}
