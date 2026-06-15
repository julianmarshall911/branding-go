// Copyright 2024 Julian Marshall
// Licensed under the Apache License, Version 2.0

package branding

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"regexp"
	"strings"
)

// AdaptLogoForBackground adjusts a logo so it's visible on a dark nav background.
// For SVGs: parses the XML and replaces dark fill/stroke colors with light ones.
// For raster images: per-pixel brightness adjustment.
// Returns the adapted logo as a data URL.
func AdaptLogoForBackground(logoDataURL, navBgHex string) (string, error) {
	navBg, err := ParseHex(navBgHex)
	if err != nil {
		return logoDataURL, nil
	}
	navGray := grayscale(navBg.R, navBg.G, navBg.B)

	// Skip adaptation if nav background is light — logo is likely already visible
	if navGray > 60 {
		return logoDataURL, nil
	}

	// SVGs need XML-level color replacement (can't do pixel manipulation)
	if strings.Contains(logoDataURL, "image/svg") {
		return adaptSVGForBackground(logoDataURL, navGray)
	}

	// Decode raster data URL
	img, err := decodeDataURL(logoDataURL)
	if err != nil {
		return logoDataURL, nil // can't adapt, return original
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 4 || h < 4 {
		return logoDataURL, nil
	}

	// Convert to NRGBA for pixel manipulation
	nrgba := image.NewNRGBA(bounds)
	draw.Draw(nrgba, bounds, img, bounds.Min, draw.Src)

	// Detect logo background color from corners and edge midpoints
	logoBg := detectBgColor(nrgba)

	// If background detected, find content bounds and crop
	if logoBg != nil {
		cb := findContentBounds(nrgba, logoBg)
		pad := 2
		cropX := max(0, cb.left-pad)
		cropY := max(0, cb.top-pad)
		cropRight := min(w, cb.right+pad+1)
		cropBottom := min(h, cb.bottom+pad+1)
		cropRect := image.Rect(cropX, cropY, cropRight, cropBottom)
		cropped := image.NewNRGBA(image.Rect(0, 0, cropRect.Dx(), cropRect.Dy()))
		draw.Draw(cropped, cropped.Bounds(), nrgba, cropRect.Min, draw.Src)
		nrgba = cropped
		bounds = nrgba.Bounds()
	}

	// Adapt pixels: if a pixel's brightness is too close to the nav background,
	// invert its lightness so it becomes visible
	threshold := 0.10 * 255 // ~25.5
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := nrgba.NRGBAAt(x, y)
			if c.A < 10 {
				continue // skip transparent
			}
			pxGray := grayscale(c.R, c.G, c.B)
			if math.Abs(pxGray-navGray) < threshold {
				// Replace with inverted lightness
				lightGray := uint8(math.Max(0, math.Min(255, 255-pxGray)))
				nrgba.SetNRGBA(x, y, color.NRGBA{R: lightGray, G: lightGray, B: lightGray, A: c.A})
			}
		}
	}

	// Encode as PNG data URL
	var buf bytes.Buffer
	if err := png.Encode(&buf, nrgba); err != nil {
		return logoDataURL, nil
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/png;base64," + encoded, nil
}

// adaptSVGForBackground parses SVG XML and replaces dark fill/stroke colors
// with lighter versions, preserving colors that already have good contrast.
func adaptSVGForBackground(logoDataURL string, navGray float64) (string, error) {
	// Extract SVG content from data URL
	comma := strings.Index(logoDataURL, ",")
	if comma < 0 {
		return logoDataURL, nil
	}
	header := logoDataURL[5:comma]
	svgBytes, err := base64.StdEncoding.DecodeString(logoDataURL[comma+1:])
	if err != nil {
		return logoDataURL, nil
	}
	svgStr := string(svgBytes)

	threshold := 255 * 0.10 // brightness threshold

	// Replace hex colors in fill and stroke attributes
	replaceColor := func(hex string) string {
		c, err := ParseHex(hex)
		if err != nil {
			return hex
		}
		pxGray := grayscale(c.R, c.G, c.B)
		if math.Abs(pxGray-navGray) < threshold {
			// Too close to nav background — invert lightness
			lightGray := uint8(math.Max(0, math.Min(255, 255-pxGray)))
			return fmt.Sprintf("#%02x%02x%02x", lightGray, lightGray, lightGray)
		}
		return hex // enough contrast, keep original
	}

	// Replace colors in fill="..." and stroke="..." attributes
	svgStr = reSVGFill.ReplaceAllStringFunc(svgStr, func(match string) string {
		return reHexInAttr.ReplaceAllStringFunc(match, func(hex string) string {
			return replaceColor(hex)
		})
	})

	// Replace colors in style="..." attributes (fill:, stroke:, color:)
	svgStr = reSVGStyle.ReplaceAllStringFunc(svgStr, func(match string) string {
		return reHexInAttr.ReplaceAllStringFunc(match, func(hex string) string {
			return replaceColor(hex)
		})
	})

	// Handle SVGs that use "black" or "Black" as named color
	svgStr = strings.ReplaceAll(svgStr, `fill="black"`, `fill="`+replaceColor("#000000")+`"`)
	svgStr = strings.ReplaceAll(svgStr, `fill="Black"`, `fill="`+replaceColor("#000000")+`"`)
	svgStr = strings.ReplaceAll(svgStr, `stroke="black"`, `stroke="`+replaceColor("#000000")+`"`)

	// Handle SVGs with no explicit fill (default is black) — add fill to root <svg>
	// Only if the SVG doesn't already have a fill on the root element
	if !strings.Contains(svgStr[:min(500, len(svgStr))], "fill=") {
		svgStr = strings.Replace(svgStr, "<svg", `<svg fill="`+replaceColor("#000000")+`"`, 1)
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(svgStr))
	return "data:" + header + "," + encoded, nil
}

var (
	reSVGFill   = regexp.MustCompile(`(?i)(?:fill|stroke)\s*=\s*"[^"]*"`)
	reHexInAttr = regexp.MustCompile(`#[0-9a-fA-F]{3,6}\b`)
	reSVGStyle  = regexp.MustCompile(`(?i)style\s*=\s*"[^"]*"`)
)

func grayscale(r, g, b uint8) float64 {
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

func decodeDataURL(dataURL string) (image.Image, error) {
	// Parse "data:image/png;base64,XXXX"
	if !strings.HasPrefix(dataURL, "data:") {
		return nil, fmt.Errorf("not a data URL")
	}
	comma := strings.Index(dataURL, ",")
	if comma < 0 {
		return nil, fmt.Errorf("invalid data URL")
	}
	header := dataURL[5:comma] // "image/png;base64"
	data, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	reader := bytes.NewReader(data)
	if strings.Contains(header, "png") {
		return png.Decode(reader)
	}
	if strings.Contains(header, "jpeg") || strings.Contains(header, "jpg") {
		return jpeg.Decode(reader)
	}
	// Try generic decode
	img, _, err := image.Decode(reader)
	return img, err
}

// detectBgColor samples corners and edge midpoints to find a consistent background color.
func detectBgColor(img *image.NRGBA) *color.NRGBA {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Sample 8 points: 4 corners + 4 edge midpoints
	tl := img.NRGBAAt(bounds.Min.X, bounds.Min.Y)

	points := []image.Point{
		{bounds.Min.X, bounds.Min.Y},                   // top-left
		{bounds.Max.X - 1, bounds.Min.Y},               // top-right
		{bounds.Min.X, bounds.Max.Y - 1},               // bottom-left
		{bounds.Max.X - 1, bounds.Max.Y - 1},           // bottom-right
		{bounds.Min.X + w/2, bounds.Min.Y},              // top-center
		{bounds.Min.X + w/2, bounds.Max.Y - 1},          // bottom-center
		{bounds.Min.X, bounds.Min.Y + h/2},              // left-center
		{bounds.Max.X - 1, bounds.Min.Y + h/2},          // right-center
	}

	for _, p := range points {
		c := img.NRGBAAt(p.X, p.Y)
		if !colorMatch(tl, c, 15) {
			return nil // not consistent
		}
	}

	return &tl
}

func colorMatch(a, b color.NRGBA, tolerance uint8) bool {
	if a.A < 10 && b.A < 10 {
		return true // both transparent
	}
	if a.A < 10 || b.A < 10 {
		return false // one transparent
	}
	return absDiff(a.R, b.R) <= tolerance &&
		absDiff(a.G, b.G) <= tolerance &&
		absDiff(a.B, b.B) <= tolerance
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

type contentBounds struct {
	top, bottom, left, right int
}

// findContentBounds scans inward from edges to find where non-background content starts.
func findContentBounds(img *image.NRGBA, bg *color.NRGBA) contentBounds {
	bounds := img.Bounds()
	cb := contentBounds{
		top:    bounds.Min.Y,
		bottom: bounds.Max.Y - 1,
		left:   bounds.Min.X,
		right:  bounds.Max.X - 1,
	}

	// Scan from top
topScan:
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !colorMatch(*bg, img.NRGBAAt(x, y), 20) {
				cb.top = y
				break topScan
			}
		}
	}

	// Scan from bottom
bottomScan:
	for y := bounds.Max.Y - 1; y >= cb.top; y-- {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !colorMatch(*bg, img.NRGBAAt(x, y), 20) {
				cb.bottom = y
				break bottomScan
			}
		}
	}

	// Scan from left
leftScan:
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := cb.top; y <= cb.bottom; y++ {
			if !colorMatch(*bg, img.NRGBAAt(x, y), 20) {
				cb.left = x
				break leftScan
			}
		}
	}

	// Scan from right
rightScan:
	for x := bounds.Max.X - 1; x >= cb.left; x-- {
		for y := cb.top; y <= cb.bottom; y++ {
			if !colorMatch(*bg, img.NRGBAAt(x, y), 20) {
				cb.right = x
				break rightScan
			}
		}
	}

	return cb
}
