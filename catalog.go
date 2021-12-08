package downloadosm

import (
	"fmt"
	"log"
	"sync"
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

func (c *Catalog) Init(client *Client) {

	fmt.Println("retrieving available files")

	wg := sync.WaitGroup{}
	wg.Add(len(c.Mirrors))

	for _, m := range c.Mirrors {
		m.Verbose = c.Verbose
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

	// tsToHash := getAttrToHash(sourcesWithHash, "timestamp", "file date")
	// if tsToHash == nil {
	// 	return nil, fmt.Errorf("unable to consistently load date - dates do not match")
	// }

	// lenToHash := getAttrToHash(sourcesWithHash, "file_len", "length")
	// if lenToHash != nil || len(sourcesWithoutHash) == 0 {
	// 	for _, src := range sourcesWithoutHash {
	// 		// if src.FileLength
	// 		_ = src
	// 	}
	// }
}

// func getTimestampToHash(sourcesWithHash map[string][]*Source) map[string]bool {
// 	h := make(map[string]bool)
// 	for _, sources := range sourcesWithHash {
// 		for _, source := range sources {
// 			ts := source.Timestamp
// 			if ts.IsZero() {
// 				continue
// 			}
// 			hash, ok := attrToHash[attr]
// 			if !ok {
// 				attrToHash[attr] = source.Hash
// 				continue
// 			}
// 			if hash != source.Hash {
// 				log.Printf("error: multiple files with the same %s have different hashes:\n", attrDesc)
// 				log.Printf("* %v, %s=%s, hash=%s\n", source, attrDesc, attr, source.Hash)
// 				src := sourcesWithHash[attrToHash[attr]][0]
// 				v := getAttr(src, attrName)
// 				attr := v.String()
// 				log.Printf("* %v, %s=%s, hash=%s\n", src, attrDesc, attr, src.Hash)
// 				return nil
// 			}
// 		}
// 	}
// 	return attrToHash
// }

// func getAttr(obj interface{}, name string) reflect.Value {
// 	p := reflect.ValueOf(obj)
// 	s := p.Elem()
// 	if s.Kind() != reflect.Struct {
// 		panic("not struct")
// 	}
// 	f := s.FieldByName(name)
// 	if !f.IsValid() {
// 		panic("not found:" + name)
// 	}
// 	return f
// }
