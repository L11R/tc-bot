package database

import (
	"database/sql"
	"tc-bot/model"
)

func (i *Instance) InsertUser(user *model.User) (*model.User, error) {
	rows, err := i.db.NamedQuery(
		`INSERT INTO users (telegram_id) VALUES (:telegram_id) RETURNING *`,
		user,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var result model.User
		if err := rows.StructScan(&result); err != nil {
			return nil, err
		}

		return &result, nil
	}

	return nil, sql.ErrNoRows
}

func (i *Instance) GetUser(user *model.User) (*model.User, error) {
	var result model.User
	err := i.db.Get(
		&result,
		`SELECT * FROM users WHERE id = $1 OR telegram_id = $2`,
		user.ID,
		user.TelegramID,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (i *Instance) UpdateState(user *model.User) error {
	_, err := i.db.Exec(
		`UPDATE users SET state = $3 WHERE id = $1 OR telegram_id = $2`,
		user.ID,
		user.TelegramID,
		user.State,
	)
	if err != nil {
		return err
	}

	return nil
}

func (i *Instance) UpdateCurrentCardID(user *model.User) error {
	_, err := i.db.Exec(
		`UPDATE users SET current_card_id = $3 WHERE id = $1 OR telegram_id = $2`,
		user.ID,
		user.TelegramID,
		user.CurrentCardID,
	)
	if err != nil {
		return err
	}

	return nil
}

func (i *Instance) UpdateCurrentFormID(user *model.User) error {
	_, err := i.db.Exec(
		`UPDATE users SET current_form_id = $3 WHERE id = $1 OR telegram_id = $2`,
		user.ID,
		user.TelegramID,
		user.CurrentFormID,
	)
	if err != nil {
		return err
	}

	return nil
}
