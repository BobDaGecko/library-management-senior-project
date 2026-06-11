package common

import (
	"fmt"
	"time"

	"voxelprismatic/library-management-senior-project/db"
)

// patronNavStatus returns overdue loan count and outstanding fine balance for
// the logged-in patron. Both are zero for guests (user == nil), so guest
// page loads pay no DB overhead.
func patronNavStatus(user *db.UserPartial) (overdueCount int64, fineBalance float32) {
	if user == nil {
		return
	}

	// user.ID is the short base64 form; the user_id columns store hex.
	uid, err := db.ParseShort(user.ID)
	if err != nil {
		return
	}

	db.Db().Model(&db.Loan{}).
		Where("user_id = ?", uid).
		Where("date_returned = ?", db.NilTime).
		Where("date_checkout < ?", time.Now().Add(-db.LOAN_DURATION)).
		Count(&overdueCount)

	db.Db().Table("fines").
		Joins("JOIN loans ON loans.id = fines.loan_id").
		Where("loans.user_id = ?", uid).
		Where("fines.amount_remaining > 0").
		Select("COALESCE(SUM(fines.amount_remaining), 0)").
		Scan(&fineBalance)

	return
}

func formatBalance(v float32) string {
	return fmt.Sprintf("$%.2f", v)
}

func formatOverdueCount(n int64) string {
	if n == 1 {
		return "1 overdue"
	}
	return fmt.Sprintf("%d overdue", n)
}
