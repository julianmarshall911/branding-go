// Copyright 2024 Julian Marshall
// Licensed under the Apache License, Version 2.0

// Package branding extracts logos, favicons, and brand colors from websites.
//
// Given a URL, it fetches the page HTML, detects the site's logo,
// extracts accent colors from CSS, and derives a full color palette
// suitable for theming a dark-mode UI.
//
// Usage:
//
//	result, err := branding.Extract("https://example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Palette.Primary)  // e.g. "#3b82f6"
//	fmt.Println(result.LogoDataURL)       // base64 PNG data URL
package branding

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Result holds everything extracted from a website.
type Result struct {
	// LogoDataURL is the logo image as a base64 PNG data URL, or empty if not found.
	LogoDataURL string `json:"logo_data_url,omitempty"`
	// FaviconURL is the absolute URL of the site's favicon.
	FaviconURL string `json:"favicon_url,omitempty"`
	// Palette is the derived color palette from the site's branding.
	Palette Palette `json:"palette"`
}

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// Extract fetches a website and extracts its logo, favicon, and brand colors.
func Extract(siteURL string) (*Result, error) {
	// Normalize URL
	if !strings.HasPrefix(siteURL, "http://") && !strings.HasPrefix(siteURL, "https://") {
		siteURL = "https://" + siteURL
	}

	parsed, err := url.Parse(siteURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Fetch HTML
	req, err := http.NewRequest("GET", siteURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", siteURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2MB limit
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	html := string(body)

	result := &Result{}
	origin := parsed.Scheme + "://" + parsed.Host

	// Extract favicon
	result.FaviconURL = extractFavicon(html, origin)

	// Detect logo
	logoURL := detectLogo(html, origin)
	if logoURL != "" {
		if dataURL, err := downloadAsDataURL(logoURL); err == nil {
			result.LogoDataURL = dataURL
		}
	}

	// Extract accent color from CSS first (preferred over logo)
	accent := ExtractAccentFromHTML(html)
	if accent != "" {
		result.Palette = DerivePalette(accent)
	} else {
		result.Palette = DefaultPalette()
	}

	return result, nil
}

// extractFavicon finds the best favicon URL from HTML.
func extractFavicon(html, origin string) string {
	// Try apple-touch-icon first (higher res)
	if href := findLinkHref(html, "apple-touch-icon"); href != "" {
		return resolveURL(href, origin)
	}
	// Then standard icon
	if href := findLinkHref(html, "icon"); href != "" {
		return resolveURL(href, origin)
	}
	// Fallback
	return origin + "/favicon.ico"
}

// downloadAsDataURL fetches an image and returns it as a base64 data URL.
// Returns empty string if the image is too large (>200KB) or fails.
func downloadAsDataURL(imageURL string) (string, error) {
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 200*1024)) // 200KB limit
	if err != nil {
		return "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	// Normalize content type
	if strings.Contains(contentType, "svg") {
		contentType = "image/svg+xml"
	} else if strings.Contains(contentType, "png") {
		contentType = "image/png"
	} else if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
		contentType = "image/jpeg"
	} else if strings.Contains(contentType, "webp") {
		contentType = "image/webp"
	} else if strings.Contains(contentType, "ico") {
		contentType = "image/x-icon"
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return "data:" + contentType + ";base64," + encoded, nil
}

// resolveURL makes a potentially relative URL absolute.
func resolveURL(href, origin string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		return origin + href
	}
	return origin + "/" + href
}
