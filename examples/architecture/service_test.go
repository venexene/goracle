package architecture

import (
	"context"
	"errors"
	"testing"
)

type fakeAccounts struct {
	debitErr error
	debits   int
	credits  int
}

func (f *fakeAccounts) Debit(context.Context, int64, int64) error {
	f.debits++
	return f.debitErr
}
func (f *fakeAccounts) Credit(context.Context, int64, int64) error {
	f.credits++
	return nil
}

type fakeUnitOfWork struct{ accounts *fakeAccounts }

func (f fakeUnitOfWork) WithinTransaction(ctx context.Context, fn func(Accounts) error) error {
	return fn(f.accounts)
}

func TestTransferUsesOneTransactionBoundary(t *testing.T) {
	accounts := &fakeAccounts{}
	service := NewTransferService(fakeUnitOfWork{accounts})
	if err := service.Transfer(t.Context(), 1, 2, 100); err != nil {
		t.Fatal(err)
	}
	if accounts.debits != 1 || accounts.credits != 1 {
		t.Fatalf("debits=%d credits=%d", accounts.debits, accounts.credits)
	}
}

func TestTransferStopsAfterDebitError(t *testing.T) {
	accounts := &fakeAccounts{debitErr: ErrInsufficientFunds}
	service := NewTransferService(fakeUnitOfWork{accounts})
	if err := service.Transfer(t.Context(), 1, 2, 100); !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("err=%v", err)
	}
	if accounts.credits != 0 {
		t.Fatal("credit must not run after debit error")
	}
}
