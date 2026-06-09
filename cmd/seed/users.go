package main

import (
	"fmt"

	"voxelprismatic/library-management-senior-project/db"
)

const (
	adminPassword  = "Admin1234!"
	sharedPassword = "Library123!"
)

// credential is a printable login row for the demo credential sheet.
type credential struct {
	email    string
	password string
	role     string
	note     string
}

// seedCast holds every account the circulation step needs to reference.
type seedCast struct {
	admin         db.User
	collaborators []db.User
	personas      map[string]db.User // keyed: clean active maxedout overdue paid holds limited locked
	filler        []db.User
	creds         []credential
}

func seedUsers() *seedCast {
	fmt.Println("Users: creating staff, personas, and patrons…")
	c := &seedCast{personas: map[string]db.User{}}

	// Super-admin (preserved credentials).
	c.admin = c.make("Library", "Admin", "admin@library.dev", adminPassword,
		db.UserRoleAdmin, db.UserStatusActive, "super-admin (preserved)")

	// Collaborators — roles per the project roles wiki.
	type collab struct {
		first, last string
		role        db.UserRoleFlag
	}
	collabs := []collab{
		{"Kellen", "Siczka", db.UserRoleAdmin},
		{"Daniel", "Petrov", db.UserRoleAdmin},
		{"Dimitar", "Dimitrov", db.UserRoleAdmin},
		{"Ahnna", "Williams", db.UserRoleAdmin},
		{"Gauri", "Aklujkar", db.UserRoleLibrarian},
		{"Allen", "Vives", db.UserRoleLibrarian},
		{"Alvar", "Kandikatla", db.UserRoleLibrarian},
	}
	for _, cl := range collabs {
		email := fmt.Sprintf("%s@library.dev", lower(cl.first))
		u := c.make(cl.first, cl.last, email, sharedPassword, cl.role, db.UserStatusActive, "collaborator")
		c.collaborators = append(c.collaborators, u)
	}

	// Designed personas (all public patrons). Note describes the demo state the
	// circulation step will build on top of them.
	type persona struct {
		key, first, last string
		status           db.UserStatusFlag
		note             string
	}
	personas := []persona{
		{"clean", "Clara", "Newcomer", db.UserStatusActive, "no activity"},
		{"active", "Aaron", "Reader", db.UserStatusActive, "3 current loans"},
		{"maxedout", "Maxine", "Bookworm", db.UserStatusActive, "7/8 loans (near limit)"},
		{"overdue", "Otto", "Latimer", db.UserStatusActive, "2 overdue + outstanding fine"},
		{"paid", "Pia", "Settled", db.UserStatusActive, "returned history + paid fine"},
		{"holds", "Hana", "Waitlist", db.UserStatusActive, "4 queued holds"},
		{"limited", "Liam", "Capped", db.UserStatusLimited, "account limited"},
		{"locked", "Lena", "Barred", db.UserStatusLocked, "account locked + fine"},
	}
	for _, p := range personas {
		email := fmt.Sprintf("%s@library.dev", p.key)
		u := c.make(p.first, p.last, email, sharedPassword, db.UserRolePublic, p.status, "persona: "+p.note)
		c.personas[p.key] = u
	}

	// Filler patrons so the directory + dashboard counts look populated.
	for _, n := range fillerNames {
		email := fmt.Sprintf("%s.%s@library.dev", lower(n.first), lower(n.last))
		u := c.make(n.first, n.last, email, sharedPassword, db.UserRolePublic, db.UserStatusActive, "")
		c.filler = append(c.filler, u)
	}

	fmt.Printf("  \x1b[32m✓ %d users\x1b[0m (1 admin, %d collaborators, %d personas, %d filler)\n",
		1+len(c.collaborators)+len(c.personas)+len(c.filler),
		len(c.collaborators), len(c.personas), len(c.filler))
	return c
}

// make builds + persists a user via the model's validated setters and records a
// credential row. Panics on failure — a broken account means a broken demo.
func (c *seedCast) make(first, last, email, pw string, role db.UserRoleFlag, status db.UserStatusFlag, note string) db.User {
	u := db.User{Roles: role, Status: status}
	if err := u.SetFirstName(first); err != nil {
		panic(err)
	}
	if err := u.SetLastName(last); err != nil {
		panic(err)
	}
	if err := u.SetEmail(email); err != nil {
		panic(fmt.Errorf("%s: %w", email, err))
	}
	if err := u.SetSecret(pw, pw); err != nil {
		panic(fmt.Errorf("%s: %w", email, err))
	}
	if err := db.Db().Create(&u).Error; err != nil {
		panic(fmt.Errorf("create %s: %w", email, err))
	}
	c.creds = append(c.creds, credential{u.Email, pw, role.DisplayName(), note})
	return u
}

func lower(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r >= 'A' && r <= 'Z' {
			out[i] = r + 32
		}
	}
	return string(out)
}

// fillerNames is a fixed roster so seeds are reproducible.
var fillerNames = []struct{ first, last string }{
	{"Marcus", "Chen"}, {"Priya", "Nair"}, {"Sofia", "Ramirez"},
	{"Jamal", "Carter"}, {"Yuki", "Tanaka"}, {"Elena", "Volkov"},
	{"Noah", "Bergstrom"}, {"Amara", "Okafor"}, {"Ravi", "Patel"},
	{"Greta", "Lindqvist"}, {"Theo", "Mwangi"}, {"Isabel", "Costa"},
	{"Hugo", "Fischer"}, {"Mei", "Lin"}, {"Diego", "Santos"},
}

// printCredentials writes the demo login sheet at the end of the run.
func printCredentials(c *seedCast) {
	fmt.Println("\n\x1b[1m── Demo credentials ──────────────────────────────────\x1b[0m")
	fmt.Printf("  %-34s %-12s %-10s %s\n", "EMAIL", "PASSWORD", "ROLE", "NOTE")
	for _, cr := range c.creds {
		fmt.Printf("  %-34s %-12s %-10s %s\n", cr.email, cr.password, cr.role, cr.note)
	}
}
