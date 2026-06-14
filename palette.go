// Copyright 2024 Julian Marshall
// Licensed under the Apache License, Version 2.0

package branding

// Palette is a complete set of UI colors derived from a primary brand color.
// Designed for dark-mode UIs with a dark navigation sidebar/topbar.
type Palette struct {
	// Primary is the main brand color.
	Primary string `json:"primary"`
	// Interactive is a lighter variant for hover states and links.
	Interactive string `json:"interactive"`
	// NavBg is the navigation background color.
	NavBg string `json:"nav_bg"`
	// NavBorder is the navigation border color.
	NavBorder string `json:"nav_border"`
	// Background is the page body background color.
	Background string `json:"background"`
	// Muted is the color for secondary/disabled text.
	Muted string `json:"muted"`
	// Success is green for positive actions.
	Success string `json:"success"`
	// Danger is red for destructive actions and errors.
	Danger string `json:"danger"`
	// Warning is amber for warnings.
	Warning string `json:"warning"`
	// Chart is a set of 10 distinct, colorblind-friendly colors for data visualization.
	Chart []string `json:"chart"`
}

type bgPreset struct {
	navBg      string
	navBorder  string
	background string
	muted      string
}

var (
	darkBlueBg = bgPreset{
		navBg:      "#192847",
		navBorder:  "#2a3a5c",
		background: "#111b33",
		muted:      "#8b9bc0",
	}
	greyBg = bgPreset{
		navBg:      "#1e1e1e",
		navBorder:  "#333333",
		background: "#141414",
		muted:      "#999999",
	}
)

// DefaultPalette returns the default palette when no brand color can be extracted.
func DefaultPalette() Palette {
	return Palette{
		Primary:     "#63b3ed",
		Interactive: "#90cdf4",
		NavBg:       darkBlueBg.navBg,
		NavBorder:   darkBlueBg.navBorder,
		Background:  darkBlueBg.background,
		Muted:       darkBlueBg.muted,
		Success:     "#48bb78",
		Danger:      "#f56565",
		Warning:     "#b8941a",
		Chart:       defaultChartColors(),
	}
}

// DerivePalette generates a complete UI palette from a primary brand color.
//
// The algorithm:
//   - If the primary color is too close to the dark blue background (RGB distance < 80
//     or similar hue with low luminance), it switches to a grey background to maintain contrast.
//   - If the primary is too dark (luminance < 0.4), it is lightened for readability.
//   - The interactive color is a lightened variant of the primary.
//   - Status colors (success, danger, warning) and chart colors are fixed for consistency.
func DerivePalette(primary string) Palette {
	c, err := ParseHex(primary)
	if err != nil {
		return DefaultPalette()
	}

	hsl := c.ToHSL()

	// Decide background theme
	dist := ColorDistance(primary, darkBlueBg.background)
	hDiff := HueDiff(primary, darkBlueBg.background)
	useGrey := dist < 80 || (hDiff < 30 && hsl.L < 0.4)

	bg := darkBlueBg
	if useGrey {
		bg = greyBg
	}

	// Lighten dark primaries for readability on dark backgrounds
	effectivePrimary := primary
	if hsl.L < 0.4 {
		effectivePrimary = Lighten(primary, 0.4-hsl.L)
	}

	interactive := Lighten(effectivePrimary, 0.15)

	return Palette{
		Primary:     effectivePrimary,
		Interactive: interactive,
		NavBg:       bg.navBg,
		NavBorder:   bg.navBorder,
		Background:  bg.background,
		Muted:       bg.muted,
		Success:     "#48bb78",
		Danger:      "#f56565",
		Warning:     "#b8941a",
		Chart:       defaultChartColors(),
	}
}

func defaultChartColors() []string {
	return []string{
		"#5ea4e0", "#e8854a", "#5ec490", "#cf6b9f", "#c4a84d",
		"#7e8ce0", "#49b6b0", "#d46a6a", "#8cc455", "#b07ed4",
	}
}
