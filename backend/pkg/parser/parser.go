package parser

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

// Common errors
var (
	ErrEmptyHTML     = errors.New("empty HTML content")
	ErrInvalidDoc    = errors.New("invalid document")
	ErrTitleNotFound = errors.New("title not found")
)

// URLHandler is a function type for handling URLs
type URLHandler func(string) string

// Document wraps goquery.Document to add custom functionality
type Document struct {
	*goquery.Document
}

// NewDocument creates a new Document from HTML string
func NewDocument(html string) (*Document, error) {
	if strings.TrimSpace(html) == "" {
		return nil, ErrEmptyHTML
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create document")
	}

	return &Document{doc}, nil
}

// ImageResult represents the result of image extraction
type ImageResult struct {
	Title string
	URLs  []string
}

// GetImages finds images with a given class name
func GetImages(html, imgClass string, urlHandler URLHandler) (*ImageResult, error) {
	doc, err := NewDocument(html)
	if err != nil {
		return nil, err
	}

	title, err := doc.ExtractTitle()
	if err != nil {
		return nil, err
	}

	urls := doc.ExtractImageURLs(imgClass, urlHandler)

	return &ImageResult{
		Title: title,
		URLs:  urls,
	}, nil
}

// ExtractImageURLs extracts image URLs with the given class
func (d *Document) ExtractImageURLs(imgClass string, urlHandler URLHandler) []string {
	var urls []string
	selector := fmt.Sprintf("img[class=\"%s\"]", imgClass)

	d.Find(selector).Each(func(i int, s *goquery.Selection) {
		if url, exists := s.Attr("src"); exists {
			if urlHandler != nil {
				url = urlHandler(url)
			}
			urls = append(urls, url)
		}
	})

	return urls
}

// ExtractTitle extracts the title from the document
func (d *Document) ExtractTitle() (string, error) {
	// Try h1 tag first
	if title := d.extractH1Title(); title != "" {
		return title, nil
	}

	// Try og:title meta tag
	if title := d.extractMetaTitle(); title != "" {
		return title, nil
	}

	// Try title tag
	if title := d.extractTitleTag(); title != "" {
		return title, nil
	}

	return "", ErrTitleNotFound
}

// Helper methods for title extraction
func (d *Document) extractH1Title() string {
	h1Elem := d.Find("h1").First()
	if title, exists := h1Elem.Attr("title"); exists {
		return cleanTitle(title)
	}
	return cleanTitle(h1Elem.Text())
}

func (d *Document) extractMetaTitle() string {
	title, _ := d.Find("meta[property=\"og:title\"]").Attr("content")
	return cleanTitle(title)
}

func (d *Document) extractTitleTag() string {
	return cleanTitle(d.Find("title").Text())
}

// cleanTitle cleans and normalizes title text
func cleanTitle(title string) string {
	return strings.TrimSpace(strings.ReplaceAll(title, "\n", ""))
}
