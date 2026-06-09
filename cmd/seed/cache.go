package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"voxelprismatic/library-management-senior-project/db"
)

// cachePath is relative to the repo root (where `make seed` / `go run` executes).
// It lives outside senior-library.db so it survives the wipe — the API is only
// ever hit on a cache miss.
const cachePath = "cmd/seed/cache/books.json"

// bookCache maps a stable lookup key (ISBN or "title|author" slug) to the fully
// resolved BookWork fetched from Google Books.
type bookCache map[string]db.BookWork

func loadCache() bookCache {
	c := bookCache{}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return c // missing/unreadable cache → start empty
	}
	if err := json.Unmarshal(data, &c); err != nil {
		fmt.Printf("  \x1b[33mwarn: cache unreadable (%v); starting fresh\x1b[0m\n", err)
		return bookCache{}
	}
	return c
}

func saveCache(c bookCache) {
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		fmt.Printf("  \x1b[33mwarn: could not create cache dir: %v\x1b[0m\n", err)
		return
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Printf("  \x1b[33mwarn: could not marshal cache: %v\x1b[0m\n", err)
		return
	}
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		fmt.Printf("  \x1b[33mwarn: could not write cache: %v\x1b[0m\n", err)
	}
}
