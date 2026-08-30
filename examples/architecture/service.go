package architecture

import (
	"context"
	"errors"
)

var (
	ErrAlreadyTransferred = errors.New("перевод уже выполнен")
	ErrInsufficientFunds  = errors.New("недостаточно средств")
)

type Accounts interface {
	Debit(context.Context, int64, int64) error
	Credit(context.Context, int64, int64) error
}

type UnitOfWork interface {
	WithinTransaction(context.Context, func(Accounts) error) error
}

type TransferService struct{ transactions UnitOfWork }

func NewTransferService(transactions UnitOfWork) *TransferService {
	return &TransferService{transactions: transactions}
}

func (s *TransferService) Transfer(ctx context.Context, from, to, amount int64) error {
	if amount <= 0 || from == to {
		return errors.New("некорректный перевод")
	}
	return s.transactions.WithinTransaction(ctx, func(accounts Accounts) error {
		if err := accounts.Debit(ctx, from, amount); err != nil {
			return err
		}
		return accounts.Credit(ctx, to, amount)
	})
}
