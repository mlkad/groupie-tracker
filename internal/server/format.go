package server

import (
	"strconv"
	"strings"
)

func cleanDate(s string) string {
	return strings.TrimPrefix(s, "*")
}

func formatLocation(s string) string {
	parts := strings.Split(s, "-")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.ReplaceAll(part, "_", " ")
		words := strings.Fields(part)

		for i, word := range words {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}

		out = append(out, strings.Join(words, " "))
	}
	return strings.Join(out, ", ")
}

func atoiFn(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}

func selectedSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}
