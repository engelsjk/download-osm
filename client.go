package downloadosm

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	c         *http.Client
	UserAgent string
}

func NewClient(userAgent string) *Client {

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	c := &http.Client{
		Timeout:   10 * time.Second,
		Transport: t,
	}

	return &Client{
		c:         c,
		UserAgent: userAgent,
	}
}

func (c *Client) Get(url string) (string, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("received status=%s\n", resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read url=%s\n", url)
	}

	return string(b), nil
}

func (c *Client) ContentLength(url string) (int, error) {

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("status=%s\n for HEAD request", resp.Status)
	}

	return strconv.Atoi(resp.Header.Get("Content-Length"))
}
