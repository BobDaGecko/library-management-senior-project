package db

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

var _ = Migrate(BookWork{})

// A literary work with all relevant metadata
type BookWork struct {
	// This struct intentionally does not inherit the BaseModel struct
	// The ID here is supplied by Google, not our own UUID
	gorm.Model
	ID string `gorm:"primaryKey"` // Google Books Volume ID

	Title         string        // E.g. A Woman Underground
	Subtitle      string        // E.g. A Cameron Winter Mystery
	Authors       SqlStringList `gorm:"type:text"`
	Publisher     string
	PublishedDate time.Time
	Version       string // As provided by Google Books

	Isbn13 string
	Isbn10 string

	Description string
	PageCount   int
	IsMature    bool
	Categories  SqlStringList `gorm:"type:text"`

	CoverThumb string
	CoverImage string
}

func (b *BookWork) Tags() []string {
	set := map[string]bool{}
	for _, category := range b.Categories {
		for subcategory := range strings.SplitSeq(category, "/") {
			subcategory = strings.TrimSpace(subcategory)
			set[subcategory] = true
		}
	}

	ret := make([]string, len(set))
	i := 0
	for k := range set {
		ret[i] = k
		i++
	}
	return ret
}

type BookVariants map[string][]BookWork

func (v *BookVariants) Add(b BookWork) {
	id := b.Isbn13
	arr, ok := (*v)[id]
	if !ok {
		id = b.Isbn10
		arr, ok = (*v)[id]
	}
	if ok {
		for _, e := range arr {
			if e.ID == b.ID {
				// Duplicate in search for whatever reason
				return
			}
		}
		(*v)[id] = append(arr, b)
		return
	}

	id = b.Isbn13
	if id == "" {
		id = b.Isbn10
	}
	if id == "" {
		id = b.ID
	}
	(*v)[id] = []BookWork{b}
}

type CopyCount struct {
	Total     int
	Available int
}

type copyCountInter struct {
	Format BookFmtFlag
	Count  int
}

// AvailableCountsBulk returns per-format total/available copy counts for many
// works in a single grouped query. Use this instead of calling AvailableCopies
// per book when rendering result lists — the per-book version costs three
// queries each. Counts are strict (per work ID); editions are not merged.
func AvailableCountsBulk(ids []string) (map[string]FormatsMap[CopyCount], error) {
	ret := map[string]FormatsMap[CopyCount]{}
	if len(ids) == 0 {
		return ret, nil
	}

	var rows []struct {
		BookWorkID string
		Format     BookFmtFlag
		Total      int
		Available  int
	}
	err := Db().Raw(`
		SELECT book_work_id, format,
		       COUNT(*) AS total,
		       SUM(CASE WHEN NOT EXISTS (
		           SELECT 1 FROM loans l
		           WHERE l.book_copy_id = book_copies.id AND l.date_returned = ?
		       ) THEN 1 ELSE 0 END) AS available
		FROM book_copies
		WHERE status = ? AND book_work_id IN ? AND deleted_at IS NULL
		GROUP BY book_work_id, format
	`, NilTime, CopyStatusPublic, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		m, ok := ret[r.BookWorkID]
		if !ok {
			m = FormatsMap[CopyCount]{}
			ret[r.BookWorkID] = m
		}
		m[r.Format] = CopyCount{Total: r.Total, Available: r.Available}
	}
	return ret, nil
}

// ExistingBookIDs reports which of the given work IDs are already in the
// local catalog, in one query (replaces per-card Exists() calls).
func ExistingBookIDs(ids []string) map[string]bool {
	ret := map[string]bool{}
	if len(ids) == 0 {
		return ret
	}
	var found []string
	Db().Model(&BookWork{}).Where("id IN ?", ids).Pluck("id", &found)
	for _, id := range found {
		ret[id] = true
	}
	return ret
}

// Lists available copies
// If 'strict' is true, then only copies for this particular book ID will be returned
// If 'strict' is false, then copies matching the title and authors will be returned, too
func (b *BookWork) AvailableCopies(strict bool) (FormatsMap[CopyCount], error) {
	db := Db()
	ids := []string{b.ID}
	if !strict {
		err := db.Model(&BookWork{}).
			Where(&BookWork{
				Title:    b.Title,
				Subtitle: b.Subtitle,
				Authors:  b.Authors,
			}).Pluck("id", &ids).Error
		if err != nil {
			return nil, err
		}
	}

	var totalCounts []copyCountInter
	err := db.Model(&BookCopy{}).
		Where("status = ?", CopyStatusPublic).
		Where("book_work_id IN ?", ids).
		Group("format").
		Select("format, COUNT(*) as count").
		Scan(&totalCounts).Error
	if err != nil {
		return nil, err
	}

	checkedOutSubquery := db.Table("loans l").
		Select("1").
		Where("l.book_copy_id = book_copies.id").
		Where("l.date_returned = ?", NilTime)

	var availableCounts []copyCountInter
	err = db.Model(&BookCopy{}).
		Where("status = ?", CopyStatusPublic).
		Where("book_work_id IN ?", ids).
		Where("NOT EXISTS (?)", checkedOutSubquery).
		Group("format").
		Select("format, COUNT(*) as count").
		Scan(&availableCounts).Error
	if err != nil {
		return nil, err
	}

	ret := FormatsMap[CopyCount]{}
	for _, tc := range totalCounts {
		ret[tc.Format] = CopyCount{Total: tc.Count}
	}
	for _, ac := range availableCounts {
		if c, ok := ret[ac.Format]; ok {
			c.Available = ac.Count
			ret[ac.Format] = c
		} else {
			ret[ac.Format] = CopyCount{
				Total:     ac.Count,
				Available: ac.Count,
			}
		}
	}

	return ret, nil
}
