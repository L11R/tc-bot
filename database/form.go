package database

import (
	"database/sql"
	"tc-bot/model"
)

func (i *Instance) InsertForm(form *model.Form) (*model.Form, error) {
	rows, err := i.db.NamedQuery(
		`INSERT INTO forms (card_id, view_state, event_validation, captcha_link) VALUES (:card_id, :view_state, :event_validation, :captcha_link) RETURNING *`,
		form,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var result model.Form
		if err := rows.StructScan(&result); err != nil {
			return nil, err
		}

		return &result, nil
	}

	return nil, sql.ErrNoRows
}

func (i *Instance) GetForm(form *model.Form) (*model.Form, error) {
	var result model.Form
	err := i.db.Get(
		&result,
		`SELECT * FROM forms WHERE id = $1 AND card_id = $2`,
		form.ID,
		form.CardID,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
