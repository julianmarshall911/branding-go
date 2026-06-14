# branding-go

A Go package that extracts logos, favicons, and brand colors from any website URL. Given a URL, it fetches the page, detects the site's logo, extracts accent colors from CSS, and derives a complete dark-mode UI color palette.

Pure Go — no CGO, no external dependencies beyond the standard library.

## Install

```bash
go get github.com/julianmarshall911/branding-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    branding "github.com/julianmarshall911/branding-go"
)

func main() {
    result, err := branding.Extract("https://stripe.com")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Logo:", result.LogoDataURL[:50], "...")
    fmt.Println("Favicon:", result.FaviconURL)
    fmt.Println("Primary color:", result.Palette.Primary)
    fmt.Println("Nav background:", result.Palette.NavBg)
}
```

## What It Does

### Logo Detection

Finds the site logo using a priority-ordered search:

1. **JSON-LD** `schema.org` logo field (most authoritative)
2. **`<a>` tags** with "logo" in class/id containing an `<img>`
3. **`<img>` tags** with "logo" in class, id, or alt attributes
4. **First `<img>` in `<header>`**
5. **`og:image`** meta tag (fallback)

Tiny icons (flags, language selectors, 16x16/24x24) are automatically filtered out. Lazy-loaded images (`data-src`) are preferred over placeholder `src` attributes.

The logo is downloaded and returned as a base64 data URL (max 200KB).

### Favicon Detection

Finds the best favicon in order: `apple-touch-icon` → `<link rel="icon">` → `/favicon.ico` fallback.

### Color Extraction

Extracts the dominant brand color from the page's CSS:

- Scans `background-color`, `background`, `color`, and `border-color` properties for hex colors
- Boosts CSS custom properties named `--primary`, `--brand`, `--accent`, `--main` (weighted 10x)
- Scores each color by **saturation × frequency**
- Penalizes red hues (reds signal danger in UI, not branding)
- Filters out near-black (lightness < 15%), near-white (> 85%), and greys (saturation < 25%)

### Palette Derivation

From a single primary brand color, generates a complete 10-field palette for dark-mode UIs:

| Field | Purpose |
|-------|---------|
| `Primary` | Main brand color |
| `Interactive` | Lighter variant for hover states |
| `NavBg` | Navigation background |
| `NavBorder` | Navigation border |
| `Background` | Page body background |
| `Muted` | Secondary/disabled text |
| `Success` | Green for positive actions |
| `Danger` | Red for errors/destructive actions |
| `Warning` | Amber for warnings |
| `Chart` | 10 colorblind-friendly chart colors |

**Smart background selection:** If the primary color is too close to the default dark-blue background (RGB distance < 80 or similar hue with low luminance), the palette switches to a grey background to maintain contrast.

**Dark color handling:** Primary colors with luminance below 40% are automatically lightened for readability on dark backgrounds.

## API Reference

### Main

```go
// Extract fetches a website and extracts its logo, favicon, and brand colors.
func Extract(siteURL string) (*Result, error)
```

### Color Utilities

```go
// ExtractAccentFromHTML finds the most prominent brand color in HTML/CSS.
func ExtractAccentFromHTML(html string) string

// DerivePalette generates a complete UI palette from a primary hex color.
func DerivePalette(primary string) Palette

// DefaultPalette returns the default palette (blue theme).
func DefaultPalette() Palette
```

### Color Conversion

```go
func ParseHex(hex string) (RGB, error)    // "#RRGGBB" or "#RGB" → RGB
func (c RGB) ToHex() string               // RGB → "#rrggbb"
func (c RGB) ToHSL() HSL                  // RGB → HSL
func HSLToRGB(h, s, l float64) RGB        // HSL → RGB
func HSLToHex(h, s, l float64) string     // HSL → "#rrggbb"
func Lighten(hex string, amount float64) string
func Darken(hex string, amount float64) string
func ColorDistance(a, b string) float64    // Euclidean RGB distance
func HueDiff(a, b string) float64         // Absolute hue difference in degrees
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).

Copyright 2024 Julian Marshall.
