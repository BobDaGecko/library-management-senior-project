package main

import (
	"fmt"

	"voxelprismatic/library-management-senior-project/db"
)

// numZeroAvailable popular titles are forced to have zero available copies (all
// checked out) so holds have something to queue behind.
const numZeroAvailable = 10

// seedCopies creates physical copies for every book with a weighted-random
// spread of counts, formats, conditions, and statuses. It returns the copies
// grouped by BookWork.ID and the set of book IDs deliberately left 0-available
// (their copies are all kept Public here so the circulation step can loan them
// out completely).
func seedCopies(books []db.BookWork) (map[string][]db.BookCopy, map[string]bool) {
	fmt.Println("Copies: generating physical inventory…")

	// Choose the zero-available demo titles from books that will get >=2 copies.
	zeroAvailable := map[string]bool{}
	for _, idx := range rng.Perm(len(books)) {
		if len(zeroAvailable) >= numZeroAvailable {
			break
		}
		zeroAvailable[books[idx].ID] = true
	}

	copiesByBook := map[string][]db.BookCopy{}
	total := 0

	for _, book := range books {
		forceAvailable := zeroAvailable[book.ID]
		n := weightedCopyCount()
		if forceAvailable && n < 2 {
			n = 2 // ensure the "fully checked out" hook is visible
		}
		formats := pickFormats()

		for i := 0; i < n; i++ {
			status := db.CopyStatusPublic
			if !forceAvailable {
				status = weightedStatus()
			}
			copy := db.BookCopy{
				BookWorkID: book.ID,
				Format:     formats[rng.Intn(len(formats))],
				Condition:  weightedCondition(),
				Status:     status,
			}
			if err := db.Db().Create(&copy).Error; err != nil {
				fmt.Printf("  \x1b[33mwarn: copy create failed for %q: %v\x1b[0m\n", book.Title, err)
				continue
			}
			copiesByBook[book.ID] = append(copiesByBook[book.ID], copy)
			total++
		}
	}

	fmt.Printf("  \x1b[32m✓ %d copies\x1b[0m across %d books (%d titles forced 0-available)\n",
		total, len(books), len(zeroAvailable))
	return copiesByBook, zeroAvailable
}

// weightedCopyCount returns 1-5, biased toward 2-3.
func weightedCopyCount() int {
	r := rng.Intn(100)
	switch {
	case r < 20:
		return 1
	case r < 50:
		return 2
	case r < 75:
		return 3
	case r < 90:
		return 4
	default:
		return 5
	}
}

// pickFormats returns a non-empty subset of formats, biased toward print.
func pickFormats() []db.BookFmtFlag {
	candidates := []struct {
		fmt  db.BookFmtFlag
		prob int // out of 100
	}{
		{db.BookFmtPaperback, 80},
		{db.BookFmtHardCover, 60},
		{db.BookFmtDigitalBook, 40},
		{db.BookFmtDigitalAudio, 30},
		{db.BookFmtPhysicalAudio, 15},
	}
	var out []db.BookFmtFlag
	for _, c := range candidates {
		if rng.Intn(100) < c.prob {
			out = append(out, c.fmt)
		}
	}
	if len(out) == 0 {
		out = append(out, db.BookFmtPaperback)
	}
	return out
}

// weightedCondition: ~70% mint/good, ~20% fair, ~10% poor/dead/lost.
func weightedCondition() db.ConditionFlag {
	r := rng.Intn(100)
	switch {
	case r < 25:
		return db.ConditionMint
	case r < 70:
		return db.ConditionGood
	case r < 90:
		return db.ConditionFair
	case r < 96:
		return db.ConditionPoor
	case r < 98:
		return db.ConditionDead
	default:
		return db.ConditionLost
	}
}

// weightedStatus: mostly public, some repairing, a few discarded.
func weightedStatus() db.CopyStatusFlag {
	r := rng.Intn(100)
	switch {
	case r < 88:
		return db.CopyStatusPublic
	case r < 96:
		return db.CopyStatusRepairing
	default:
		return db.CopyStatusDiscarded
	}
}
