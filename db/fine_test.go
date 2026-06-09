package db

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestMarkFinePaidZerosBalance(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}
	tx.Save(&user)

	fine := Fine{
		UserID:          user.ID,
		IssueReason:     FineReasonLate,
		IssueDate:       time.Now().Add(-DAY),
		AmountIssued:    5.0,
		AmountRemaining: 5.0,
	}
	tx.Create(&fine)

	tx.Model(&Fine{}).Where("id = ?", fine.ID).Update("amount_remaining", float32(0))

	var updated Fine
	tx.Where("id = ?", fine.ID).First(&updated)
	assert.Equal(t, updated.AmountRemaining, float32(0))
}

func TestMarkFinePaidCreatesTransaction(t *testing.T) {
	tx := TestDb()
	defer tx.Rollback()

	user := User{
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}
	tx.Save(&user)

	fine := Fine{
		UserID:          user.ID,
		IssueReason:     FineReasonLate,
		IssueDate:       time.Now().Add(-DAY),
		AmountIssued:    7.5,
		AmountRemaining: 7.5,
	}
	tx.Create(&fine)

	txn := Transaction{
		UserID:     fine.UserID,
		AmountPaid: fine.AmountRemaining,
		Date:       time.Now(),
	}
	tx.Create(&txn)
	tx.Model(&Fine{}).Where("id = ?", fine.ID).Update("amount_remaining", float32(0))

	var txns []Transaction
	tx.Where("user_id = ?", user.ID).Find(&txns)
	assert.Equal(t, len(txns), 1)
	assert.Equal(t, txns[0].AmountPaid, float32(7.5))
}
