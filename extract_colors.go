// Copyright 2026 Julian Marshall
// Licensed under the Apache License, Version 2.0

package branding

import (
	"math"
	"sort"
)

// BrandColors holds all significant colors extracted from a website,
// sorted by prominence. No filtering — all colors are preserved for
// context-aware palette derivation.
type BrandColors struct {
	// Colors sorted by score (most prominent first)
	Colors []ScoredColor `json:"colors"`
}

// ScoredColor is a color with its extraction score.
type ScoredColor struct {
	Hex   string  `json:"hex"`
	Score float64 `json:"score"`
	Hue   float64 `json:"hue"`
	Sat   float64 `json:"sat"`
	Lum   float64 `json:"lum"`
}

// ExtractAllColors extracts all significant colors from HTML/CSS, unbiased.
// No color is filtered out — blacks, whites, reds are all included.
// Scoring is based on saturation × frequency, with CSS variable boosts.
func ExtractAllColors(html string) BrandColors {
	cleaned := reWPGradient.ReplaceAllString(html, "")
	cleaned = reWPVar.ReplaceAllString(cleaned, "")

	counts := make(map[string]int)
	for _, m := range reCSSColor.FindAllStringSubmatch(cleaned, -1) {
		hex := normalizeHex(m[1])
		if hex != "" {
			counts[hex]++
		}
	}
	for _, m := range reBrandVar.FindAllStringSubmatch(cleaned, -1) {
		hex := normalizeHex(m[1])
		if hex != "" {
			counts[hex] += 10
		}
	}

	var colors []ScoredColor
	for hex, freq := range counts {
		c, err := ParseHex(hex)
		if err != nil {
			continue
		}
		hsl := c.ToHSL()

		// Score by saturation × frequency (vivid, common colors rank higher)
		score := hsl.S * float64(freq)
		// Boost mid-range lightness (most useful for UI)
		if hsl.L > 0.2 && hsl.L < 0.8 {
			score *= 1.5
		}

		colors = append(colors, ScoredColor{
			Hex:   hex,
			Score: score,
			Hue:   hsl.H,
			Sat:   hsl.S,
			Lum:   hsl.L,
		})
	}

	sort.Slice(colors, func(i, j int) bool {
		return colors[i].Score > colors[j].Score
	})

	return BrandColors{Colors: colors}
}

// ForDarkBackground returns the best colors for a dark background.
// Filters out colors that would be invisible (too dark, too desaturated).
func (bc *BrandColors) ForDarkBackground() []ScoredColor {
	var result []ScoredColor
	for _, c := range bc.Colors {
		// Skip very dark desaturated colors (invisible on dark bg)
		if c.Lum < 0.2 && c.Sat < 0.3 {
			continue
		}
		result = append(result, c)
	}
	return result
}

// ForLightBackground returns the best colors for a light background.
// Filters out colors that would be invisible (too light, too desaturated).
func (bc *BrandColors) ForLightBackground() []ScoredColor {
	var result []ScoredColor
	for _, c := range bc.Colors {
		// Skip very light desaturated colors (invisible on light bg)
		if c.Lum > 0.8 && c.Sat < 0.3 {
			continue
		}
		result = append(result, c)
	}
	return result
}

// Primary returns the top color for a given background mode.
func (bc *BrandColors) Primary(dark bool) string {
	var candidates []ScoredColor
	if dark {
		candidates = bc.ForDarkBackground()
	} else {
		candidates = bc.ForLightBackground()
	}
	if len(candidates) == 0 {
		if dark {
			return "#63b3ed"
		}
		return "#3182ce"
	}
	return candidates[0].Hex
}

// TopN returns the top N distinct colors (by hue distance) for a background mode.
// Ensures visual variety — won't return 3 shades of blue.
func (bc *BrandColors) TopN(n int, dark bool) []string {
	var candidates []ScoredColor
	if dark {
		candidates = bc.ForDarkBackground()
	} else {
		candidates = bc.ForLightBackground()
	}

	const minHueDistance = 30.0
	var selected []ScoredColor
	for _, c := range candidates {
		tooClose := false
		for _, s := range selected {
			if hueDist(c.Hue, s.Hue) < minHueDistance && math.Abs(c.Lum-s.Lum) < 0.2 {
				tooClose = true
				break
			}
		}
		if !tooClose {
			selected = append(selected, c)
			if len(selected) >= n {
				break
			}
		}
	}

	result := make([]string, len(selected))
	for i, c := range selected {
		result[i] = c.Hex
	}
	return result
}

func hueDist(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}
