package downloadosm

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

type Source struct {
	Name       string
	URL        string
	Timestamp  time.Time
	Mirror     *Mirror
	URLHash    string
	Hash       string
	FileLength int64
	Verbose    bool
}

type SourceOption func(*Source)

func WithSourceName(name string) SourceOption {
	return func(s *Source) {
		s.Name = name
	}
}

func WithSourceURL(url string) SourceOption {
	return func(s *Source) {
		s.URL = url
	}
}

func WithSourceTimestamp(timestamp time.Time) SourceOption {
	return func(s *Source) {
		s.Timestamp = timestamp
	}
}

func WithSourceMirror(mirror *Mirror) SourceOption {
	return func(s *Source) {
		s.Mirror = mirror
	}
}

func WithSourceVerbose(verbose bool) SourceOption {
	return func(s *Source) {
		s.Verbose = verbose
	}
}

//////////////////////////////////

func NewSource(opts ...SourceOption) *Source {
	s := &Source{}
	for _, opt := range opts {
		opt(s)
	}
	if s.URLHash == "" {
		s.URLHash = s.URL + ".md5"
	}
	return s
}

func (s *Source) String() string {
	return fmt.Sprintf(`%s from %s`, s.Name, s.URL)
}

func (s *Source) StringHash() string {
	return fmt.Sprintf(`%s from %s`, s.Name, s.URLHash)
}

func (s *Source) LoadHash(client *Client) error {

	ErrUnableToLoadHash := fmt.Errorf("unable to load md5 hash for %s", s.StringHash())

	if s.URLHash == "" {
		return fmt.Errorf("%w: no source url hash", ErrUnableToLoadHash)
	}

	if s.Verbose {
		log.Printf("getting md5 checksum from %s\n", s.URLHash)
	}

	r, err := client.Get(s.URLHash)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrUnableToLoadHash, err.Error())
	}

	hash := (strings.Split(strings.TrimSpace(r), " "))[0]

	if matched, _ := regexp.MatchString(`^[a-fA-F0-9]{32}$`, hash); !matched {
		return fmt.Errorf("%w: invalid md5 hash %s", ErrUnableToLoadHash, hash)
	}

	s.Hash = hash

	return nil
}

func (s *Source) LoadMetadata(client *Client) error {

	ErrUnableToLoadMetadata := fmt.Errorf("unable to load metadata for %s", s.Name)

	if s.URL == "" {
		return fmt.Errorf("%w: no source url", ErrUnableToLoadMetadata)
	}

	if s.Verbose {
		log.Printf("getting content length for %s\n", s.URL)
	}

	fileLength, err := client.ContentLength(s.URL)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrUnableToLoadMetadata, err.Error())
	}

	s.FileLength = int64(fileLength)

	return nil
}

func (s *Source) Size() string {
	if s.FileLength == 0 {
		return "Unknown"
	}
	return fmt.Sprintf("%d MB (%s)", s.FileLength/1024/1024, humanize.Comma(s.FileLength))
}

func printSources(sources []*Source) {
	for _, s := range sources {
		log.Printf("%s %s %s %s %d\n", s.Name, s.URL, s.Hash, s.Timestamp, s.FileLength)
	}
}
