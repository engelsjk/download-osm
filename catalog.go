package downloadosm

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
)

type Catalog struct {
	Mirrors     []*Mirror
	UsePrimary  bool
	ForceLatest bool
	Verbose     bool
}

type CatalogOption func(*Catalog)

func WithCatalogMirrors(mirrors []*Mirror) CatalogOption {
	return func(c *Catalog) {
		c.Mirrors = mirrors
	}
}

func WithCatalogUsePrimary() CatalogOption {
	return func(c *Catalog) {
		c.UsePrimary = true
	}
}

func WithCatalogForceLatest() CatalogOption {
	return func(c *Catalog) {
		c.ForceLatest = true
	}
}

func WithCatalogVerbose() CatalogOption {
	return func(c *Catalog) {
		c.Verbose = true
	}
}

func NewCatalog(opts ...CatalogOption) (*Catalog, error) {
	c := &Catalog{}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

type Stat struct {
	Timestamp   time.Time
	MirrorCount int
	Hash        string
	Size        string
}

func (s Stat) String() string {
	return fmt.Sprintf("hash: %s | mirrors: %d | timestamp: %s | size: %s", s.Hash, s.MirrorCount, s.Timestamp, s.Size)
}

type Downloads struct {
	URLs []string
	Hash string
}

func (d Downloads) SaveURLs() {
	file, err := os.Create("urls.tsv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Comma = '\t'
	writer.WriteAll([][]string{d.URLs})
}

func (c *Catalog) Init(client *Client) (*Downloads, error) {

	log.Println("retrieving available files")

	wg := sync.WaitGroup{}
	wg.Add(len(c.Mirrors))

	for _, m := range c.Mirrors {
		m.Verbose = c.Verbose
		if c.Verbose {
			log.Printf("initializing mirror %s from %s\n", m.URL, m.Country)
		}
		go func(m *Mirror) {
			defer wg.Done()
			if err := m.Init(client); err != nil {
				log.Println(err.Error())
			}
		}(m)
	}
	wg.Wait()

	sourcesWithHash := make(map[string][]*Source)
	sourcesWithoutHash := []*Source{}

	for _, m := range c.Mirrors {
		for _, s := range m.Sources {
			if s.Hash == "" {
				sourcesWithoutHash = append(sourcesWithoutHash, s)
				continue
			}
			sourcesWithHash[s.Hash] = append(sourcesWithHash[s.Hash], s)
		}
	}

	timestampToHash := getTimestampToHash(sourcesWithHash)
	if timestampToHash == nil {
		return nil, fmt.Errorf("unable to consistently load date - dates do not match")
	}

	fileLengthToHash := getFileLengthToHash(sourcesWithHash)
	if fileLengthToHash != nil || len(sourcesWithoutHash) == 0 { // debug: how does the 2nd conditional here work with the range below?
		for _, source := range sourcesWithoutHash {
			if hash, ok := fileLengthToHash[source.FileLength]; ok {
				sourcesWithHash[hash] = append(sourcesWithHash[hash], source)
				continue
			}
			fl := "unknown"
			if source.FileLength != 0 {
				fl = humanize.Comma(source.FileLength)
			}
			log.Printf("source %v has unrecognized file length=%s\n", source, fl)
		}
	} else {
		log.Println(`unable to use sources - unable to match "latest" without date/md5:`)
		for _, source := range sourcesWithoutHash {
			log.Println(source.String())
		}
	}

	stats := []*Stat{}

	for _, sources := range sourcesWithHash {

		// debug: double check this is sorting by (1) timestamp then (2) filelength
		sort.Slice(sources, func(i, j int) bool {
			timestamp := sources[i].Timestamp
			if timestamp.IsZero() {
				timestamp = time.Unix(1<<63-1, 0)
			}
			if timestamp.After(sources[j].Timestamp) {
				return true
			}
			if timestamp.Before(sources[j].Timestamp) {
				return false
			}
			return sources[i].FileLength < sources[j].FileLength
		})

		stats = append(stats, &Stat{
			Timestamp:   sources[0].Timestamp,
			MirrorCount: len(sources),
			Hash:        sources[0].Hash,
			Size:        sources[0].Size(),
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		timestamp := stats[i].Timestamp
		if timestamp.IsZero() {
			timestamp = time.Unix(1<<63-1, 0)
		}
		return timestamp.After(stats[j].Timestamp)
	})

	log.Println("latest available files:")
	for _, s := range stats {
		log.Printf("%+v\n", s) // debug: clean up print
	}

	var hashToDownload, info string
	if !c.ForceLatest && len(stats) > 1 && float64(stats[0].MirrorCount)*1.5 < float64(stats[0].MirrorCount) {
		hashToDownload = stats[1].Hash
		info = fmt.Sprintf(" because the latest %s is not widespread yet", stats[0].Timestamp)
	} else {
		hashToDownload = stats[0].Hash
		info = ""
	}

	sourceList, ok := sourcesWithHash[hashToDownload]
	if !ok {
		return nil, fmt.Errorf("something went wrong with hash %s selected to download\n", hashToDownload)
	}

	ts := "latest (unknown date)"
	timestamp := sourceList[0].Timestamp
	if !timestamp.IsZero() {
		ts = fmt.Sprintf("%s", timestamp.Format("2006-01-02"))
	}

	if len(sourceList) > 2 && !c.UsePrimary {
		for i := len(sourceList) - 1; i >= 0; i-- {
			if sourceList[i].Mirror.IsPrimary {
				sourceList = append(sourceList[:i], sourceList[i+1:]...)
			}
		}
	}

	log.Printf("will download planet published on %s, size=%s, md5=%s, using %d sources%s",
		ts,
		sourceList[0].Size(),
		sourceList[0].Hash,
		len(sourceList),
		info,
	)

	if c.Verbose {
		for _, source := range sourceList {
			log.Printf("%s : %s : %s : %d\n", source.Mirror.Country, source.URL, source.Hash, source.FileLength)
		}
	}

	downloads := &Downloads{Hash: hashToDownload}
	downloads.URLs = []string{}
	for _, source := range sourceList {
		downloads.URLs = append(downloads.URLs, source.URL)
	}

	return downloads, nil
}

func getTimestampToHash(sourcesWithHash map[string][]*Source) map[time.Time]string {
	timestampToHash := make(map[time.Time]string)
	for _, sources := range sourcesWithHash {
		for _, source := range sources {
			timestamp := source.Timestamp
			if timestamp.IsZero() {
				continue
			}
			if _, ok := timestampToHash[timestamp]; !ok {
				timestampToHash[timestamp] = source.Hash
				continue
			}
			if hash := timestampToHash[timestamp]; hash != source.Hash {
				log.Printf("multiple files with the same timestamp have different hashes:\n")
				log.Printf("* %v, file date=%s, hash=%s\n", source, timestamp, source.Hash)
				if srcs, ok := sourcesWithHash[hash]; ok {
					src := srcs[0]
					log.Printf("* %v, file date=%s, hash=%s\n", src, src.Timestamp, src.Hash)
				}
				return nil
			}
		}
	}
	return timestampToHash
}

func getFileLengthToHash(sourcesWithHash map[string][]*Source) map[int64]string {
	fileLengthToHash := make(map[int64]string)
	for _, sources := range sourcesWithHash {
		for _, source := range sources {
			fileLength := source.FileLength
			log.Printf("%s from %s at %d\n", source.Name, source.URL, fileLength)
			if fileLength == 0 {
				continue
			}
			if _, ok := fileLengthToHash[fileLength]; !ok {
				fileLengthToHash[fileLength] = source.Hash
				continue
			}
			if hash := fileLengthToHash[fileLength]; hash != source.Hash {
				log.Printf("multiple files with the same file length have different hashes:\n")
				log.Printf("* %v, file length=%d, hash=%s\n", source, fileLength, source.Hash)
				if srcs, ok := sourcesWithHash[hash]; ok {
					src := srcs[0]
					log.Printf("* %v, file length=%d, hash=%s\n", src, src.FileLength, src.Hash)
				}
				return nil
			}
		}
	}
	return fileLengthToHash
}
