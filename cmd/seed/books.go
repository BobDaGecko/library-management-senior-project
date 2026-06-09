package main

import (
	"fmt"
	"strings"
	"time"

	"voxelprismatic/library-management-senior-project/db"
	"voxelprismatic/library-management-senior-project/fetch"
)

// apiThrottle paces requests under Google Books' ~100 req/100s limit. One search
// per book (the search result already carries description + thumbnail), so ~1.1s
// keeps us comfortably below the ceiling on the first (uncached) run.
const apiThrottle = 1100 * time.Millisecond

// flushEvery persists the cache periodically so an interrupted run keeps its
// progress and a re-run resumes from where it left off.
const flushEvery = 15

// seedBooks resolves the curated booklist into BookWork rows. Each entry is
// looked up once via the Google Books search API (ISBN first, then a precise
// title+author query), and cached to disk so subsequent seeds are offline.
func seedBooks() []db.BookWork {
	fmt.Printf("Books: resolving %d curated titles…\n", len(booklist))

	cache := loadCache()
	sinceFlush := 0

	var (
		books   []db.BookWork
		seenID  = map[string]bool{}
		misses  int
		fetched int
	)

	for i, sb := range booklist {
		key := sb.cacheKey()

		book, ok := cache[key]
		if !ok {
			resolved, found := resolveBook(sb)
			if !found {
				misses++
				fmt.Printf("  \x1b[33m✗ skip\x1b[0m %s — no Google Books match\n", sb.Title)
				continue
			}
			book = resolved
			cache[key] = book
			fetched++
			if sinceFlush++; sinceFlush >= flushEvery {
				saveCache(cache)
				sinceFlush = 0
			}
		}

		if book.ID == "" || seenID[book.ID] {
			continue // unresolved or duplicate volume across entries
		}
		seenID[book.ID] = true
		books = append(books, book)

		if (i+1)%25 == 0 {
			fmt.Printf("  …%d/%d processed (%d ok, %d skipped)\n", i+1, len(booklist), len(books), misses)
		}
	}
	saveCache(cache)

	// Insert into the (freshly migrated) DB.
	for i := range books {
		if err := db.Db().Create(&books[i]).Error; err != nil {
			fmt.Printf("  \x1b[33mwarn: failed to insert %q: %v\x1b[0m\n", books[i].Title, err)
		}
	}

	fmt.Printf("  \x1b[32m✓ %d books\x1b[0m (%d fetched from API, %d from cache, %d skipped)\n",
		len(books), fetched, len(books)-fetched, misses)
	return books
}

// resolveBook turns a curated entry into a fully-populated BookWork.
// Lookup order: exact ISBN (canonical, zero bloat) → precise title+author query.
func resolveBook(sb seedBook) (db.BookWork, bool) {
	var item *fetch.GBooksVolDetails

	if sb.ISBN != "" {
		item = searchFirst("isbn:" + sb.ISBN)
	}
	if item == nil {
		q := fmt.Sprintf("intitle:%q", sb.Title)
		if sb.Author != "" {
			q += fmt.Sprintf(" inauthor:%q", sb.Author)
		}
		item = searchFirst(q)
	}
	if item == nil || item.ID == "" {
		return db.BookWork{}, false
	}

	book := item.ToLocalStruct()
	if strings.TrimSpace(book.Title) == "" {
		return db.BookWork{}, false
	}
	if book.CoverImage == "" { // search results often omit the large cover
		book.CoverImage = book.CoverThumb
	}
	return book, true
}

// searchFirst runs a search and returns the first result. It throttles before
// each call and retries with backoff so a transient rate-limit (429, which comes
// back as an empty result) doesn't permanently drop an otherwise-good title.
func searchFirst(query string) *fetch.GBooksVolDetails {
	for attempt := 0; attempt < 3; attempt++ {
		time.Sleep(apiThrottle)
		res, err := fetch.GBooksSearch(query)
		if err == nil && res != nil && len(res.Items) > 0 {
			return &res.Items[0]
		}
		time.Sleep(time.Duration(attempt+1) * time.Second) // backoff before retry
	}
	return nil
}
