package main

import (
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestGetBestWordsShuffle(t *testing.T) {
	// Save original state
	origDict := dictionary
	origEntries := dictEntries
	origByLength := dictByLength
	t.Cleanup(func() {
		dictionary = origDict
		dictEntries = origEntries
		dictByLength = origByLength
	})

	// Reset dictionary for a controlled test environment
	dictionary = make(map[string]string)
	dictEntries = nil
	for i := range dictByLength {
		dictByLength[i] = nil
	}

	// Helper to add words
	addWord := func(orig string) {
		norm := normalizeWord(orig)
		dictionary[norm] = orig
		runesCount := utf8.RuneCountInString(norm)
		entry := DictEntry{
			original: orig,
			word:     norm,
			freq:     letterFrequency(norm),
			length:   runesCount,
		}
		dictEntries = append(dictEntries, entry)
		idx := len(dictEntries) - 1
		dictByLength[runesCount] = append(dictByLength[runesCount], idx)
	}

	// We want to test a situation where we have:
	// - 2 words of length 7 ("abcdeff", "abcdegg")
	// - 10 words of length 6 ("abcdef", "abcdeg", "abcdeh", "abcdei", "abcdej", "abcdek", "abcdel", "abcdem", "abcden", "abcdeo")
	// Available letters: "abcdefghijklmno"
	//
	// In this case:
	// - The 2 words of length 7 should ALWAYS be in the top 5 results.
	// - The remaining 3 spots in the top 5 should be filled by shuffling the 10 words of length 6.
	// Therefore, running GetBestWords multiple times should produce different subsets of length-6 words.

	addWord("abcdeff")
	addWord("abcdegg")
	
	wordsOfLen6 := []string{
		"abcdef", "abcdeg", "abcdeh", "abcdei", "abcdej",
		"abcdek", "abcdel", "abcdem", "abcden", "abcdeo",
	}
	for _, w := range wordsOfLen6 {
		addWord(w)
	}

	available := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "f", "g"}

	// Let's run it 50 times and check if we see variation in the results.
	// Also ensure "abcdeff" and "abcdegg" are always present.
	firstResult := GetBestWords(available)
	if len(firstResult) != 5 {
		t.Fatalf("Expected exactly 5 words, got %d: %v", len(firstResult), firstResult)
	}

	hasVariation := false
	for i := 0; i < 50; i++ {
		res := GetBestWords(available)
		
		// Verify length-7 words are always included
		hasF := false
		hasG := false
		for _, w := range res {
			if w == "abcdeff" {
				hasF = true
			}
			if w == "abcdegg" {
				hasG = true
			}
		}
		if !hasF || !hasG {
			t.Errorf("Expected length-7 words 'abcdeff' and 'abcdegg' to always be in the result, but got %v", res)
		}

		// Compare with firstResult to see if there is variation
		if strings.Join(res, ",") != strings.Join(firstResult, ",") {
			hasVariation = true
		}
	}

	if !hasVariation {
		t.Error("Expected variation in the selected length-6 words due to Shuffle, but got the exact same results in 50 runs")
	}
}

func TestGetBestWordsMaxFive(t *testing.T) {
	// Ensure dictionary is loaded
	if len(dictEntries) == 0 {
		t.Skip("Dictionary not loaded")
	}

	// Run against a generous letter pool that should yield many words
	available := []string{"A", "A", "B", "C", "D", "E", "E", "F", "G", "H", "I", "J", "L", "M", "N", "O", "P", "R", "S", "T", "U", "V", "Z"}
	for i := 0; i < 20; i++ {
		res := GetBestWords(available)
		if len(res) > 5 {
			t.Fatalf("GetBestWords returned %d words, max allowed is 5: %v", len(res), res)
		}
	}
}

func BenchmarkGetBestWords(b *testing.B) {
	if len(dictEntries) == 0 {
		// Load actual dictionary if not populated
		var reader io.Reader
		if len(dictionaryBytes) > 0 {
			reader = strings.NewReader(string(dictionaryBytes))
		} else {
			b.Skip("dictionaryBytes empty")
		}
		_ = LoadDictionary(reader)
	}

	available := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetBestWords(available)
	}
}
