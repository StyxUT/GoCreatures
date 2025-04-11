package main

import (
	"golang.org/x/text/width"
	"unicode"
)

// Helper function to get the next safe (single width) rune
func nextSafeRune() rune {
	if nextRuneIndex >= len(safeRunes) {
		// fallback if you run out (wrap around or panic/log)
		nextRuneIndex = 0
	}
	r := safeRunes[nextRuneIndex]
	nextRuneIndex++
	return r
}

// Helper function to build a list of single width runes
func buildSingleWidthRunePool(start, end rune) []rune {
	var runes []rune
	for r := start; r <= end; r++ {
		if unicode.IsPrint(r) && isSingleColumnRune(r) {
			runes = append(runes, r)
		}
	}
	return runes
}

// Helper function to determine if a rune is single with rune
func isSingleColumnRune(r rune) bool {
	p := width.LookupRune(r)
	k := p.Kind()
	return k == width.Neutral || k == width.EastAsianNarrow || k == width.EastAsianAmbiguous
}
