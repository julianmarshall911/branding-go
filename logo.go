// Copyright 2024 Julian Marshall
// Licensed under the Apache License, Version 2.0

package branding

import (
	"regexp"
	"strings"
)

// Logo detection patterns, ordered by priority.

var (
	// JSON-LD schema.org logo
	reJSONLD = regexp.MustCompile(`(?i)"logo"\s*:\s*\{\s*[^}]*"url"\s*:\s*"([^"]+)"`)
	// Simpler JSON-LD logo (direct string value)
	reJSONLDSimple = regexp.MustCompile(`(?i)"logo"\s*:\s*"([^"]+)"`)

	// <a> tags with "logo" in class or id, containing <img>
	reLogoLink = regexp.MustCompile(`(?is)<a[^>]*(?:class|id)\s*=\s*"[^"]*logo[^"]*"[^>]*>.*?<img[^>]*(?:src|data-src)\s*=\s*"([^"]+)"`)

	// <img> tags with "logo" in class, id, or alt
	reLogoImg = regexp.MustCompile(`(?i)<img[^>]*(?:class|id|alt)\s*=\s*"[^"]*logo[^"]*"[^>]*(?:data-src|src)\s*=\s*"([^"]+)"`)
	// Same but with src before the logo attribute
	reLogoImgAlt = regexp.MustCompile(`(?i)<img[^>]*(?:data-src|src)\s*=\s*"([^"]+)"[^>]*(?:class|id|alt)\s*=\s*"[^"]*logo[^"]*"`)

	// First <img> in <header>
	reHeaderImg = regexp.MustCompile(`(?is)<header[^>]*>.*?<img[^>]*(?:data-src|src)\s*=\s*"([^"]+)"`)

	// og:image meta tag
	reOGImage = regexp.MustCompile(`(?i)<meta[^>]*property\s*=\s*"og:image"[^>]*content\s*=\s*"([^"]+)"`)
	reOGImageAlt = regexp.MustCompile(`(?i)<meta[^>]*content\s*=\s*"([^"]+)"[^>]*property\s*=\s*"og:image"`)

	// <link> tag for favicon/icons
	reLinkIcon = regexp.MustCompile(`(?i)<link[^>]*rel\s*=\s*"([^"]*)"[^>]*href\s*=\s*"([^"]+)"`)
	reLinkIconAlt = regexp.MustCompile(`(?i)<link[^>]*href\s*=\s*"([^"]+)"[^>]*rel\s*=\s*"([^"]*)"`)

	// Tiny icon and social media patterns to reject
	reTinyIcon = regexp.MustCompile(`(?i)flag|lang-|language|social|twitter|facebook|16x16|20x20|24x24`)
)

// detectLogo finds the best logo URL from HTML using priority-ordered detection.
func detectLogo(html, origin string) string {
	// 1. JSON-LD schema.org logo
	if m := reJSONLD.FindStringSubmatch(html); len(m) > 1 {
		return resolveURL(unescapeJSONLD(m[1]), origin)
	}
	if m := reJSONLDSimple.FindStringSubmatch(html); len(m) > 1 {
		return resolveURL(unescapeJSONLD(m[1]), origin)
	}

	// 2. <a> with "logo" class/id containing <img>
	if m := reLogoLink.FindStringSubmatch(html); len(m) > 1 {
		if u := filterLogoURL(m[1], origin); u != "" {
			return u
		}
	}

	// 3. <img> with "logo" in class/id/alt
	for _, re := range []*regexp.Regexp{reLogoImg, reLogoImgAlt} {
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			if u := filterLogoURL(m[1], origin); u != "" {
				return u
			}
		}
	}

	// 4. First <img> in <header>
	if m := reHeaderImg.FindStringSubmatch(html); len(m) > 1 {
		if u := filterLogoURL(m[1], origin); u != "" {
			return u
		}
	}

	// 5. og:image (fallback)
	for _, re := range []*regexp.Regexp{reOGImage, reOGImageAlt} {
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			return resolveURL(m[1], origin)
		}
	}

	return ""
}

// filterLogoURL rejects URLs that look like tiny icons or data URIs.
func filterLogoURL(href, origin string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "data:") {
		return ""
	}
	if reTinyIcon.MatchString(href) {
		return ""
	}
	return resolveURL(href, origin)
}

// unescapeJSONLD handles escaped forward slashes in JSON-LD values.
func unescapeJSONLD(s string) string {
	return strings.ReplaceAll(s, "\\/", "/")
}

// findLinkHref finds the href of a <link> tag with the given rel value.
func findLinkHref(html, rel string) string {
	for _, m := range reLinkIcon.FindAllStringSubmatch(html, -1) {
		if len(m) > 2 && strings.Contains(m[1], rel) {
			return m[2]
		}
	}
	for _, m := range reLinkIconAlt.FindAllStringSubmatch(html, -1) {
		if len(m) > 2 && strings.Contains(m[2], rel) {
			return m[1]
		}
	}
	return ""
}
