// Copyright 2024 Julian Marshall
// Licensed under the Apache License, Version 2.0

package branding

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Color conversion and CSS accent color extraction.

var (
	// Match CSS color declarations
	reCSSColor = regexp.MustCompile(`(?i)(?:background-color|background|color|border-color)\s*:\s*(#[0-9a-fA-F]{3,6})\b`)

	// Match CSS custom property names that suggest brand colors
	reBrandVar = regexp.MustCompile(`(?i)--(?:primary|brand|accent|main)\s*:\s*(#[0-9a-fA-F]{3,6})\b`)

	// WordPress noise to strip
	reWPGradient = regexp.MustCompile(`(?i)\.has-[\w-]+-gradient-background\{[^}]*\}`)
	reWPVar      = regexp.MustCompile(`(?i)--wp-[^;]+;`)
)

// RGB represents a color in the RGB color space.
type RGB struct {
	R, G, B uint8
}

// HSL represents a color in the HSL color space.
type HSL struct {
	H, S, L float64 // H: 0-360, S: 0-1, L: 0-1
}

// ParseHex parses a hex color string (#RGB or #RRGGBB) into RGB.
func ParseHex(hex string) (RGB, error) {
	hex = strings.TrimPrefix(hex, "#")

	// Expand shorthand (#ABC -> #AABBCC)
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return RGB{}, fmt.Errorf("invalid hex color: #%s", hex)
	}

	var r, g, b uint8
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return RGB{}, fmt.Errorf("parse hex %s: %w", hex, err)
	}
	return RGB{r, g, b}, nil
}

// ToHex converts RGB to a hex string.
func (c RGB) ToHex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

// ToHSL converts RGB to HSL.
func (c RGB) ToHSL() HSL {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l := (max + min) / 2

	if max == min {
		return HSL{0, 0, l}
	}

	d := max - min
	var s float64
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}

	var h float64
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h /= 6

	return HSL{h * 360, s, l}
}

// HSLToRGB converts HSL to RGB.
func HSLToRGB(h, s, l float64) RGB {
	h = h / 360.0 // normalize to 0-1

	if s == 0 {
		v := uint8(math.Round(l * 255))
		return RGB{v, v, v}
	}

	a := s * math.Min(l, 1-l)
	f := func(n float64) uint8 {
		k := math.Mod(n+h*12, 12)
		v := l - a*math.Max(math.Min(math.Min(k-3, 9-k), 1), -1)
		return uint8(math.Round(255 * math.Max(0, math.Min(1, v))))
	}
	return RGB{f(0), f(8), f(4)}
}

// HSLToHex converts HSL values to a hex color string.
func HSLToHex(h, s, l float64) string {
	return HSLToRGB(h, s, l).ToHex()
}

// Darken reduces the lightness of a hex color by amount (0-1).
func Darken(hex string, amount float64) string {
	c, err := ParseHex(hex)
	if err != nil {
		return hex
	}
	hsl := c.ToHSL()
	hsl.L = math.Max(0, hsl.L-amount)
	return HSLToHex(hsl.H, hsl.S, hsl.L)
}

// Lighten increases the lightness of a hex color by amount (0-1).
func Lighten(hex string, amount float64) string {
	c, err := ParseHex(hex)
	if err != nil {
		return hex
	}
	hsl := c.ToHSL()
	hsl.L = math.Min(1, hsl.L+amount)
	return HSLToHex(hsl.H, hsl.S, hsl.L)
}

// ColorDistance returns the Euclidean distance between two hex colors in RGB space.
func ColorDistance(a, b string) float64 {
	ca, err1 := ParseHex(a)
	cb, err2 := ParseHex(b)
	if err1 != nil || err2 != nil {
		return 999
	}
	dr := float64(ca.R) - float64(cb.R)
	dg := float64(ca.G) - float64(cb.G)
	db := float64(ca.B) - float64(cb.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// HueDiff returns the absolute hue difference in degrees between two hex colors.
func HueDiff(a, b string) float64 {
	ca, err1 := ParseHex(a)
	cb, err2 := ParseHex(b)
	if err1 != nil || err2 != nil {
		return 999
	}
	ha := ca.ToHSL().H
	hb := cb.ToHSL().H
	return math.Abs(ha - hb)
}

// ContrastColor returns "#000000" or "#ffffff" based on WCAG luminance,
// for readable text on a colored background.
func ContrastColor(hex string) string {
	c, err := ParseHex(hex)
	if err != nil {
		return "#ffffff"
	}
	lum := (0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)) / 255
	if lum > 0.5 {
		return "#000000"
	}
	return "#ffffff"
}

// isRedHue returns true if the hue is in the red range (danger zone for UI).
func isRedHue(h float64) bool {
	return h <= 30 || h >= 330
}

// ExtractAccentFromHTML finds the most prominent brand color in HTML/CSS.
// Returns a hex color string, or empty if none found.
func ExtractAccentFromHTML(html string) string {
	// Strip WordPress noise
	cleaned := reWPGradient.ReplaceAllString(html, "")
	cleaned = reWPVar.ReplaceAllString(cleaned, "")

	// Collect CSS colors with frequency counts
	counts := make(map[string]int)

	for _, m := range reCSSColor.FindAllStringSubmatch(cleaned, -1) {
		hex := normalizeHex(m[1])
		if hex != "" {
			counts[hex]++
		}
	}

	// Boost CSS custom properties (--primary, --brand, --accent, --main)
	for _, m := range reBrandVar.FindAllStringSubmatch(cleaned, -1) {
		hex := normalizeHex(m[1])
		if hex != "" {
			counts[hex] += 10
		}
	}

	// Score each color
	var bestHex string
	var bestScore float64

	for hex, freq := range counts {
		c, err := ParseHex(hex)
		if err != nil {
			continue
		}
		hsl := c.ToHSL()

		// Skip near-black, near-white, and low-saturation
		if hsl.L < 0.15 || hsl.L > 0.85 {
			continue
		}
		if hsl.S < 0.25 {
			continue
		}

		score := hsl.S * float64(freq)
		if isRedHue(hsl.H) {
			score *= 0.1 // penalize reds
		}

		if score > bestScore {
			bestScore = score
			bestHex = hex
		}
	}

	return bestHex
}

// normalizeHex expands shorthand hex and lowercases.
func normalizeHex(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	hex = strings.ToLower(hex)
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return ""
	}
	return "#" + hex
}
