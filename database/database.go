package database

import (
	"github.com/jmoiron/sqlx"
	"tc-bot/model"
)

type IDatabase interface {
	InsertUser(user *model.User) (*model.User, error)
	GetUser(user *model.User) (*model.User, error)
	UpdateState(user *model.User) error
	UpdateCurrentCardID(user *model.User) error
	UpdateCurrentFormID(user *model.User) error

	InsertCard(card *model.Card) (*model.Card, error)
	GetCard(card *model.Card) (*model.Card, error)
	SelectCardsByUserID(userID int) ([]*model.Card, error)
	DeleteCard(card *model.Card) error

	InsertForm(form *model.Form) (*model.Form, error)
	GetForm(form *model.Form) (*model.Form, error)
}

type Instance struct {
	db *sqlx.DB
}

func NewInstance(db *sqlx.DB) *Instance {
	return &Instance{db: db}
}
