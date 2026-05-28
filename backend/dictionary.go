package main

import (
	"bufio"
	_ "embed"
	"io"
	"math/rand/v2"
	"strings"
	"unicode/utf8"
)

//go:embed assets/diccionario.txt
var dictionaryBytes []byte

const AlphabetSize = 27

type DictEntry struct {
	original string
	word     string
	freq     [AlphabetSize]int
	length   int
}

var (
	dictionary   = make(map[string]string)
	dictEntries  []DictEntry
	dictByLength [11][]int
	normReplacer *strings.Replacer
	asciiIndex   [128]int
)

func init() {
	normReplacer = strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u",
	)
	for i := range asciiIndex {
		asciiIndex[i] = -1
	}
	for c := 'a'; c <= 'z'; c++ {
		asciiIndex[c] = int(c - 'a')
	}
}

func normalizeWord(w string) string {
	return normReplacer.Replace(strings.ToLower(w))
}

func runeIndex(r rune) int {
	if r < 128 {
		return asciiIndex[r]
	}
	if r == 'ñ' {
		return 26
	}
	return -1
}

// LoadDictionary carga el diccionario desde un io.Reader
func LoadDictionary(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	seen := make(map[string]bool)

	for scanner.Scan() {
		w := strings.TrimSpace(scanner.Text())
		if w == "" {
			continue
		}
		norm := normalizeWord(w)
		
		valid := true
		for _, r := range norm {
			if runeIndex(r) < 0 {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}

		if !seen[norm] {
			seen[norm] = true
			dictionary[norm] = w
			runesCount := len([]rune(norm))
			if runesCount >= 5 && runesCount <= 10 {
				entry := DictEntry{
					original: w,
					word:     norm,
					freq:     letterFrequency(norm),
					length:   runesCount,
				}
				dictEntries = append(dictEntries, entry)
			}
		}
	}

	for i := range dictByLength {
		dictByLength[i] = nil
	}
	for idx, entry := range dictEntries {
		dictByLength[entry.length] = append(dictByLength[entry.length], idx)
	}

	return scanner.Err()
}

// IsValidWord comprueba si la palabra existe
func IsValidWord(word string) (bool, string) {
	norm := normalizeWord(word)
	orig, ok := dictionary[norm]
	return ok, orig
}

// IsConstructible comprueba si se puede formar target a partir de available
func IsConstructible(target string, available []string) bool {
	// Contar letras disponibles
	var counts [AlphabetSize]int
	for _, l := range available {
		for _, r := range normalizeWord(l) {
			if idx := runeIndex(r); idx >= 0 {
				counts[idx]++
			}
		}
	}

	// Restar letras necesarias
	for _, r := range target {
		idx := runeIndex(r)
		if idx < 0 || counts[idx] == 0 {
			return false
		}
		counts[idx]--
	}
	return true
}

// GetBestWords devuelve las 5 mejores palabras que se pueden formar
func GetBestWords(available []string) []string {
	var normalized []string
	for _, l := range available {
		normalized = append(normalized, normalizeWord(l))
	}
	availableFreq := letterFrequency(strings.Join(normalized, ""))

	var result []string
	minLength := -1

	for length := 10; length >= 5; length-- {
		if len(result) >= 5 && length < minLength {
			break
		}

		for _, idx := range dictByLength[length] {
			entry := dictEntries[idx]
			if len(result) >= 5 && entry.length < minLength {
				break
			}

			valid := true
			for i := 0; i < AlphabetSize; i++ {
				if entry.freq[i] > availableFreq[i] {
					valid = false
					break
				}
			}
			if valid {
				result = append(result, entry.original)
				minLength = entry.length
			}
		}
	}

	if len(result) > 5 {
		var top, tied []string
		for _, w := range result {
			if utf8.RuneCountInString(w) > minLength {
				top = append(top, w)
			} else {
				tied = append(tied, w)
			}
		}
		needed := 5 - len(top)
		if needed > 0 {
			rand.Shuffle(len(tied), func(i, j int) { tied[i], tied[j] = tied[j], tied[i] })
			result = append(top, tied[:needed]...)
		} else {
			result = top
		}
	}
	return result
}

func letterFrequency(s string) [AlphabetSize]int {
	var freq [AlphabetSize]int
	for _, r := range s {
		if idx := runeIndex(r); idx >= 0 {
			freq[idx]++
		}
	}
	return freq
}
