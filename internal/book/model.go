package book

import "time"

type Book struct {
	ID, Title, Format string
	Chapters          []Chapter
}
type Chapter struct{ ID, Title, Volume, Text string }
type Position struct {
	ChapterID string
	Offset    int
}
type Metadata struct {
	ID, Title, Format, CurrentChapter string
	Progress                          float64
	ImportedAt                        time.Time
}
