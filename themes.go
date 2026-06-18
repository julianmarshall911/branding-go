// Copyright 2026 Julian Marshall
// Licensed under the Apache License, Version 2.0

package branding

// DualPalette contains both dark and light mode palettes derived from the same brand colors.
type DualPalette struct {
	Dark  Palette `json:"dark"`
	Light Palette `json:"light"`
}

var (
	lightBlueBg = bgPreset{
		navBg:      "#f0f4f8",
		navBorder:  "#d2dae4",
		background: "#ffffff",
		muted:      "#64748b",
	}
	lightGreyBg = bgPreset{
		navBg:      "#f5f5f5",
		navBorder:  "#e0e0e0",
		background: "#ffffff",
		muted:      "#6b7280",
	}
)

// DeriveThemes generates both dark and light palettes from extracted brand colors.
func DeriveThemes(bc *BrandColors) DualPalette {
	darkPrimary := bc.Primary(true)
	lightPrimary := bc.Primary(false)

	darkPalette := deriveDarkPalette(darkPrimary, bc)
	lightPalette := deriveLightPalette(lightPrimary, bc)

	return DualPalette{
		Dark:  darkPalette,
		Light: lightPalette,
	}
}

func deriveDarkPalette(primary string, bc *BrandColors) Palette {
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

	// Lighten dark primaries for readability
	effectivePrimary := primary
	if hsl.L < 0.4 {
		effectivePrimary = Lighten(primary, 0.4-hsl.L)
	}

	interactive := Lighten(effectivePrimary, 0.15)

	// Use extracted brand colors for chart series if available
	chartColors := bc.TopN(10, true)
	if len(chartColors) < 5 {
		chartColors = defaultChartColors()
	}
	// Pad to 10 if needed
	for len(chartColors) < 10 {
		chartColors = append(chartColors, defaultChartColors()[len(chartColors)])
	}

	return Palette{
		Primary:         effectivePrimary,
		PrimaryContrast: ContrastColor(effectivePrimary),
		Interactive:     interactive,
		NavBg:           bg.navBg,
		NavBorder:       bg.navBorder,
		Background:      bg.background,
		Muted:           bg.muted,
		Success:         "#48bb78",
		Danger:          "#f56565",
		Warning:         "#b8941a",
		Chart:           chartColors,
	}
}

func deriveLightPalette(primary string, bc *BrandColors) Palette {
	c, err := ParseHex(primary)
	if err != nil {
		return DefaultLightPalette()
	}

	hsl := c.ToHSL()

	// Decide background theme
	dist := ColorDistance(primary, lightBlueBg.navBg)
	hDiff := HueDiff(primary, lightBlueBg.navBg)
	useGrey := dist < 80 || (hDiff < 30 && hsl.L > 0.6)

	bg := lightBlueBg
	if useGrey {
		bg = lightGreyBg
	}

	// Darken light primaries for readability on light backgrounds
	effectivePrimary := primary
	if hsl.L > 0.6 {
		effectivePrimary = Darken(primary, hsl.L-0.6)
	}

	interactive := Darken(effectivePrimary, 0.1)

	// Use extracted brand colors for chart series if available
	chartColors := bc.TopN(10, false)
	if len(chartColors) < 5 {
		chartColors = defaultLightChartColors()
	}
	for len(chartColors) < 10 {
		chartColors = append(chartColors, defaultLightChartColors()[len(chartColors)])
	}

	return Palette{
		Primary:         effectivePrimary,
		PrimaryContrast: ContrastColor(effectivePrimary),
		Interactive:     interactive,
		NavBg:           bg.navBg,
		NavBorder:       bg.navBorder,
		Background:      bg.background,
		Muted:           bg.muted,
		Success:         "#38a169",
		Danger:          "#e53e3e",
		Warning:         "#d69e2e",
		Chart:           chartColors,
	}
}

// DefaultLightPalette returns the default light-mode palette.
func DefaultLightPalette() Palette {
	return Palette{
		Primary:         "#3182ce",
		PrimaryContrast: "#ffffff",
		Interactive:     "#2b6cb0",
		NavBg:           lightBlueBg.navBg,
		NavBorder:       lightBlueBg.navBorder,
		Background:      lightBlueBg.background,
		Muted:           lightBlueBg.muted,
		Success:         "#38a169",
		Danger:          "#e53e3e",
		Warning:         "#d69e2e",
		Chart:           defaultLightChartColors(),
	}
}

func defaultLightChartColors() []string {
	return []string{
		"#3182ce", "#e05a2b", "#2f855a", "#b83280", "#b7791f",
		"#5a67d8", "#2c7a7b", "#c53030", "#68a039", "#805ad5",
	}
}
