package downloadosm

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anaskhan96/soup"
)

type Mirror struct {
	Country       string
	URL           string
	IsPrimary     bool
	Sources       []*Source
	HasMultiFiles bool
	Verbose       bool
}

type MirrorOption func(*Mirror)

func WithMirrorCountry(country string) MirrorOption {
	return func(m *Mirror) {
		m.Country = country
	}
}

func WithMirrorURL(url string) MirrorOption {
	return func(m *Mirror) {
		m.URL = url
	}
}

func WithMirrorPrimary() MirrorOption {
	return func(m *Mirror) {
		m.IsPrimary = true
	}
}

func WithMirrorMultiFiles() MirrorOption {
	return func(m *Mirror) {
		m.HasMultiFiles = true
	}
}

func WithMirrorVerbose(verbose bool) MirrorOption {
	return func(m *Mirror) {
		m.Verbose = verbose
	}
}

func NewMirror(opts ...MirrorOption) *Mirror {
	m := &Mirror{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Mirror) Init(client *Client) error {

	sources, err := m.GetSources()
	if err != nil {
		return fmt.Errorf("unable to use %s source %s: %w", m.Country, m.URL, err)
	}

	if len(sources) == 0 {
		return fmt.Errorf("unable to use %s source %s: no sources found", m.Country, m.URL)
	}

	m.Sources = sources

	m.LoadSources(client)

	if len(m.Sources) > 1 && m.Sources[0].Hash == m.Sources[1].Hash {
		fmt.Println("latest is the same as the last one")
		m.Sources = m.Sources[1:]
	}

	return nil
}

func (m *Mirror) GetSources() ([]*Source, error) {

	if m.HasMultiFiles {
		return m.ExtractSources()
	}

	return []*Source{
		NewSource(
			WithSourceName("planet-latest.osm.pbf"),
			WithSourceURL(m.URL),
			WithSourceMirror(m),
		),
	}, nil
}

type Link struct {
	Name string
	Href string
}

func (m *Mirror) ExtractSources() ([]*Source, error) {

	rexpPlanet := regexp.MustCompile(`^planet-(\d{6}|latest)\.osm\.pbf(\.md5)?$`)

	allSources := make(map[string]*Source)
	links := []Link{}

	resp, err := soup.Get(m.URL)
	if err != nil {
		return nil, err
	}

	doc := soup.HTMLParse(resp)
	anchors := doc.FindAll("a")

	for _, a := range anchors {
		href, ok := a.Attrs()["href"]
		if !ok {
			continue
		}
		href = strings.TrimSpace(href)
		text := strings.TrimSpace(a.Text())
		links = append(links, Link{Name: text, Href: href})
	}

	sort.Slice(links, func(i, j int) bool {
		if links[i].Name != links[j].Name {
			return links[i].Name < links[j].Name
		}
		return links[i].Href < links[j].Href
	})

	for _, link := range links {

		matched := rexpPlanet.FindStringSubmatch(link.Name)
		if len(matched) < 1 {
			if m.Verbose {
				log.Printf("ignoring unexpected name %s from %s\n", link.Name, m.URL)
			}
			continue
		}

		url := link.Href
		if !strings.Contains(link.Href, "/") {
			url = m.URL + link.Href
		}

		date := matched[1]
		dt := time.Time{}

		if date != "latest" {
			if t, err := time.Parse("20210101", date); err != nil {
				dt = t
			}
		}

		isMD5 := matched[2] == ".md5"

		if !isMD5 {
			if _, ok := allSources[date]; ok {
				log.Printf("%s already exists, while parsing %s from %s", date, link.Name, m.URL)
				continue
			}
			allSources[date] = NewSource(
				WithSourceName(link.Name),
				WithSourceURL(url),
				WithSourceTimestamp(dt),
				WithSourceMirror(m),
			)
		} else {
			if _, ok := allSources[date]; !ok {
				log.Printf("md5 file exists, but data file does not, while parsing %s from %s", link.Name, m.URL)
				continue
			}
			allSources[date].URLHash = url
		}
	}

	latest, ok := allSources["latest"]
	if ok {
		delete(allSources, "latest")
	}

	dates := make([]string, 0, len(allSources))
	for d := range allSources {
		dates = append(dates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	result := []*Source{}

	for i := 0; i < 2; i++ {
		if i == len(allSources)-1 {
			break
		}
		result = append(result, allSources[dates[i]])
	}

	if latest != nil {
		result = append([]*Source{latest}, result...)
	}

	return result, nil
}

func (m *Mirror) LoadSources(client *Client) {
	wg := sync.WaitGroup{}
	wg.Add(len(m.Sources))
	for _, s := range m.Sources {
		go func(s *Source) {
			log.Printf("load %s source %s\n", s.Name, s.URL)
			defer wg.Done()
			if err := s.LoadHash(client); err != nil {
				log.Printf(err.Error())
			}
			if err := s.LoadMetadata(client); err != nil {
				log.Printf(err.Error())
			}
		}(s)
	}
	wg.Wait()
}
