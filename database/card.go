package database

import (
	"database/sql"
	"tc-bot/model"
)

func (i *Instance) InsertCard(card *model.Card) (*model.Card, error) {
	rows, err := i.db.NamedQuery(
		`INSERT INTO cards (user_id, number) VALUES (:user_id, :number) RETURNING *`,
		card,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var result model.Card
		if err := rows.StructScan(&result); err != nil {
			return nil, err
		}

		return &result, nil
	}

	return nil, sql.ErrNoRows
}

func (i *Instance) GetCard(card *model.Card) (*model.Card, error) {
	var result model.Card
	err := i.db.Get(
		&result,
		`SELECT * FROM cards WHERE (id = $1 OR number = $3) AND user_id = $2`,
		card.ID,
		card.UserID,
		card.Number,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (i *Instance) SelectCardsByUserID(userID int) ([]*model.Card, error) {
	var result []*model.Card
	err := i.db.Select(
		&result,
		`SELECT * FROM cards WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (i *Instance) DeleteCard(card *model.Card) error {
	_, err := i.db.Exec(
		`DELETE FROM cards WHERE id = $1 AND user_id = $2`,
		card.ID,
		card.UserID,
	)
	if err != nil {
		return err
	}

	return nil
}
