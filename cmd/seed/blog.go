package main

import (
	"fmt"

	"voxelprismatic/library-management-senior-project/db"
)

// seedBlog creates a handful of staff blog posts. The home page "Staff Picks"
// section renders the 3 most recent Blog rows (falling back to recent books if
// none exist), so this is what populates that section for the demo.
func seedBlog(c *seedCast) {
	fmt.Println("Blog: staff picks…")

	// Author posts as staff members (admins/librarians).
	authors := append([]db.User{c.admin}, c.collaborators...)
	author := func(i int) db.SqlUUID { return authors[i%len(authors)].ID }

	posts := []struct {
		title, body string
		tags        []string
	}{
		{
			"Staff Picks: Cozy Reads for a Quiet Weekend",
			"Our librarians rounded up the comfort reads we keep coming back to — warm, character-driven stories perfect for an afternoon with a cup of tea.",
			[]string{"staff pick", "cozy", "fiction"},
		},
		{
			"New on the Shelf: Science Fiction Worth the Hype",
			"From first-contact epics to near-future thrillers, here are the speculative titles flying off our shelves this month.",
			[]string{"staff pick", "sci-fi", "new arrivals"},
		},
		{
			"Librarian's Choice: Nonfiction That Stays With You",
			"Big ideas, true stories, and the books our staff press into the hands of every curious patron.",
			[]string{"staff pick", "nonfiction"},
		},
		{
			"Mystery & Thriller Roundup",
			"Twisty plots and sleepless nights guaranteed. These are the page-turners our circulation desk can't keep in stock.",
			[]string{"staff pick", "mystery", "thriller"},
		},
		{
			"Fantasy Worlds to Get Lost In",
			"Sprawling sagas and standalone gems — our picks for readers who want to disappear into another world.",
			[]string{"staff pick", "fantasy"},
		},
	}

	count := 0
	for i, p := range posts {
		blog := db.Blog{
			Title:  p.title,
			Body:   p.body,
			Date:   daysAgo(len(posts) - i), // newest last-created
			Tags:   db.SqlStringList(p.tags),
			UserID: author(i),
		}
		if err := db.Db().Create(&blog).Error; err != nil {
			fmt.Printf("  \x1b[33mwarn: blog create: %v\x1b[0m\n", err)
			continue
		}
		count++
	}
	fmt.Printf("  \x1b[32m✓ %d blog posts\x1b[0m\n", count)
}
