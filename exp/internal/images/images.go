package images

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "embed"
	_ "image/png"

	"github.com/yyyoichi/httpcache-go"
	"golang.org/x/image/draw"
)

//go:embed image_urls.txt
var imageURLs []byte

// ParseURLs parses the embedded image_urls.txt and returns a slice of URLs.
func ParseURLs() []string {
	var urls []string
	scanner := bufio.NewScanner(strings.NewReader(string(imageURLs)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && strings.HasPrefix(line, "http") {
			urls = append(urls, line)
		}
	}
	return urls
}

// rateLimitedClient wraps an HTTP original client with rate limiting between requests
// Thread-safe for concurrent requests
type rateLimitedClient struct {
	client   *http.Client
	interval time.Duration
	lastCall time.Time
	mu       sync.Mutex
}

func newRateLimitedClient(interval time.Duration) *rateLimitedClient {
	return &rateLimitedClient{
		client:   http.DefaultClient,
		interval: interval,
	}
}

func (r *rateLimitedClient) Do(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Wait if needed to maintain the interval between requests
	elapsed := time.Since(r.lastCall)
	if elapsed < r.interval {
		time.Sleep(r.interval - elapsed)
	}

	resp, err := r.client.Do(req)
	r.lastCall = time.Now()

	return resp, err
}

var orignalClient = httpcache.Client{
	Client:  newRateLimitedClient(time.Duration(250 * time.Millisecond)),
	Cache:   httpcache.NewStorageCache("/tmp/pexels_http_cache/"),
	Handler: httpcache.NewDefaultHandler(),
}

type trimClient struct {
	client httpcache.Client
}

func (r *trimClient) Do(req *http.Request) (*http.Response, error) {
	// remove query parameters from the URL
	u := req.URL
	q := u.Query()
	u.RawQuery = ""
	req.URL = u
	targetWidth, err := strconv.ParseInt(q.Get("w"), 10, 64)
	if err != nil {
		return nil, err
	}
	targetHeight, err := strconv.ParseInt(q.Get("h"), 10, 64)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	src, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	srcRect := bounds
	srcRatio := float64(width) / float64(height)
	targetRatio := float64(targetWidth) / float64(targetHeight)

	if srcRatio > targetRatio {
		// source too wide - center crop
		newWidth := int(float64(height) * targetRatio)
		x := (width - newWidth) / 2
		srcRect = image.Rect(x, 0, x+newWidth, height)
	} else if srcRatio < targetRatio {
		// source too tall - center crop
		newHeight := int(float64(width) / targetRatio)
		y := (height - newHeight) / 2
		srcRect = image.Rect(0, y, width, y+newHeight)
	}

	// resize with higher quality filter
	dist := image.NewRGBA(image.Rect(0, 0, int(targetWidth), int(targetHeight)))
	draw.CatmullRom.Scale(dist, dist.Bounds(), src, srcRect, draw.Over, nil)

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, dist, &jpeg.Options{Quality: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	resp.Body = io.NopCloser(&buf)
	return resp, nil
}

var client = httpcache.Client{
	Client:  &trimClient{client: orignalClient},
	Cache:   httpcache.NewStorageCache("/tmp/pexels_http_cache/"),
	Handler: httpcache.NewDefaultHandler(),
}

// FetchImageWithSize fetches the image at the given URL and resizes/crops it to width x height.
func FetchImageWithSize(uri string, width, height int) (image.Image, error) {
	uri = getUri(uri, width, height)
	resp, err := client.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	img, err := jpeg.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode jpeg: %w", err)
	}

	return img, nil
}

func GetCachedImagePath(uri string, width, height int) string {
	u, _ := url.ParseRequestURI(getUri(uri, width, height))
	o := httpcache.NewHttpResponseObject(u)
	return "/tmp/pexels_http_cache/" + o.Key()
}

func getUri(uri string, width, height int) string {
	// Add resolution parameters
	sizeParams := fmt.Sprintf("w=%d&h=%d", width, height)
	if strings.Contains(uri, "?") {
		uri += "&" + sizeParams
	} else {
		uri += "?" + sizeParams
	}
	return uri
}
