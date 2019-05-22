package model

import (
	"database/sql"
	"github.com/lib/pq"
	"time"
)

type User struct {
	ID int `db:"id"`

	TelegramID    int           `db:"telegram_id"`
	State         sql.NullInt64 `db:"state"`
	CurrentCardID sql.NullInt64 `db:"current_card_id"`
	CurrentFormID sql.NullInt64 `db:"current_form_id"`

	Created time.Time   `db:"created"`
	Updated pq.NullTime `db:"updated"`
}

type Card struct {
	ID     int `db:"id"`
	UserID int `db:"user_id"`

	Number int `db:"number"`

	Created time.Time   `db:"created"`
	Updated pq.NullTime `db:"updated"`
}

type Form struct {
	ID     int `db:"id"`
	CardID int `db:"card_id"`

	ViewState       string `db:"view_state"`
	EventValidation string `db:"event_validation"`
	CaptchaLink     string `db:"captcha_link"`

	Created time.Time   `db:"created"`
	Updated pq.NullTime `db:"updated"`
}
