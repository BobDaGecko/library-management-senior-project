// Command seed wipes and repopulates senior-library.db with rich, demo-ready data.
//
// Run via `make seed` (which deletes the db file first and forwards the Google
// Books API token from .env). The db package auto-connects + AutoMigrates a fresh
// schema at init, so this program only needs to insert.
//
// Design + rationale: /tmp/handoff_library_db_seed_2026-06-09.md
package main

import (
	"flag"
	"fmt"
	"math/rand"

	"voxelprismatic/library-management-senior-project/fetch"
)

// rng is the single seeded source of randomness for the whole seed run, so a
// given booklist always produces the same copies/circulation spread.
var rng = rand.New(rand.NewSource(394))

func main() {
	gapiToken := flag.String("gapi-token", "", "Google Books API token (first run only; results are cached)")
	flag.Parse()
	fetch.SetAPIToken(*gapiToken)

	fmt.Println("\n\x1b[1m── Library DB seed ───────────────────────────────────\x1b[0m")

	// 1. Books (from Google Books, cached to disk).
	books := seedBooks()
	if len(books) == 0 {
		panic("no books were seeded — cannot continue (check API token / network)")
	}

	// 2. Physical copies (+ records the titles forced to 0-available for holds).
	copiesByBook, zeroAvailable := seedCopies(books)

	// 3. Users (admin, collaborators, personas, filler) — returns the cast we
	//    drive circulation against.
	cast := seedUsers()

	// 4. Circulation: loans, holds, fines, transactions.
	seedCirculation(cast, books, copiesByBook, zeroAvailable)

	// 5. Print the credential sheet for the demo.
	printCredentials(cast)

	fmt.Println("\n\x1b[32;1m✓ Seed complete.\x1b[0m Run `make run` and log in.")
}
