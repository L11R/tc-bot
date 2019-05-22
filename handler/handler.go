package handler

import (
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
	"tc-bot/config"
	"tc-bot/database"
	"tc-bot/model"
	"tc-bot/tool"
	"time"
)

type Handler struct {
	DB       database.IDatabase
	Telegram *tgbotapi.BotAPI
	Logger   logrus.FieldLogger
	Config   *config.Config
}

func NewHandler(db database.IDatabase, bot *tgbotapi.BotAPI, logger logrus.FieldLogger, conf *config.Config) *Handler {
	return &Handler{
		DB:       db,
		Telegram: bot,
		Logger:   logger,
		Config:   conf,
	}
}

func (h *Handler) Start(update tgbotapi.Update) error {
	_, err := h.DB.InsertUser(&model.User{
		TelegramID: update.Message.From.ID,
	})
	if err != nil {
		if err, ok := err.(pq.PGError); ok {
			if err.Get('C') == "23505" {
				h.Logger.WithError(err).Info("error skip")
			} else {
				return tool.NewHRError(
					"Sorry, my database seems to be down. Come later!",
					errors.Wrap(err, "cannot insert user"),
				)
			}
		} else {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot insert user"),
			)
		}
	}

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		`Hello, this bot can show balance of your transport card. Click /addcard to save your card.`,
	)

	if _, err := h.Telegram.Send(msg); err != nil {
		return errors.Wrap(err, "cannot send message")
	}

	return nil
}

func (h *Handler) AddCard(update tgbotapi.Update) error {
	if err := h.DB.UpdateState(&model.User{
		TelegramID: update.Message.From.ID,
		State: sql.NullInt64{
			Int64: 0,
			Valid: true,
		},
	}); err != nil {
		return tool.NewHRError(
			"Sorry, my database seems to be down. Come later!",
			errors.Wrap(err, "cannot update state"),
		)
	}

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		`Enter card number (usually laser engraved):`,
	)

	if _, err := h.Telegram.Send(msg); err != nil {
		return errors.Wrap(err, "cannot send message")
	}

	return nil
}

func (h *Handler) Cards(update tgbotapi.Update) error {
	user, err := h.DB.GetUser(&model.User{
		TelegramID: update.Message.From.ID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return tool.NewHRError(
				"Hm. Seems like I cannot find record about you. Try again by clicking /start!",
				errors.Wrap(err, "user somehow didn't click /start"),
			)
		} else {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot get user"),
			)
		}
	}

	cards, err := h.DB.SelectCardsByUserID(user.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return tool.NewHRError(
				"There are no cards currently. But you can always /addcard!",
				errors.Wrap(err, "cannot find cards"),
			)
		} else {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot get cards"),
			)
		}
	}

	text := "<b>Your cards:</b>\n"
	if len(cards) > 0 {
		for _, c := range cards {
			text += fmt.Sprintf("%d: /balance_%d /remove_%d\n", c.Number, c.ID, c.ID)
		}
	} else {
		text += "empty"
	}

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		text,
	)
	msg.ParseMode = "HTML"

	if _, err := h.Telegram.Send(msg); err != nil {
		return errors.Wrap(err, "cannot send message")
	}

	return nil
}

func (h *Handler) Balance(update tgbotapi.Update) error {
	user, err := h.DB.GetUser(&model.User{
		TelegramID: update.Message.From.ID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return tool.NewHRError(
				"Hm. Seems like I cannot find record about you. Try again by clicking /start!",
				errors.Wrap(err, "user somehow didn't click /start"),
			)
		} else {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot get user"),
			)
		}
	}

	if len(update.Message.Text) < strings.LastIndex(update.Message.Text, "_")+2 {
		return tool.NewHRError(
			"Incorrect command arguments.",
			errors.New("there is no card number"),
		)
	}

	cardID, err := strconv.Atoi(update.Message.Text[strings.LastIndex(update.Message.Text, "_")+1:])
	if err != nil {
		return tool.NewHRError(
			"Incorrect command arguments.",
			errors.Wrap(err, "cannot convert card number into int"),
		)
	}

	card, err := h.DB.GetCard(&model.Card{
		ID:     cardID,
		UserID: user.ID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return tool.NewHRError(
				"Hm. Seems like I cannot find record about you. Please, try again.",
				errors.Wrap(err, "cannot find card"),
			)
		} else {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot get card"),
			)
		}
	}

	if err := h.DB.UpdateCurrentCardID(&model.User{
		ID: user.ID,
		CurrentCardID: sql.NullInt64{
			Int64: int64(card.ID),
			Valid: true,
		},
	}); err != nil {
		return tool.NewHRError(
			"Sorry, my database seems to be down. Come later!",
			errors.Wrap(err, "cannot update current_card_id"),
		)
	}

	form, err := tool.GetForm()
	if err != nil {
		return tool.NewHRError(
			"Sorry, seems like balance checker service is down. Nothing to deal with it. Come later!",
			errors.Wrap(err, "cannot get form"),
		)
	}

	// Don't forget to set card_id before form insertion
	form.CardID = card.ID
	form, err = h.DB.InsertForm(form)
	if err != nil {
		return tool.NewHRError(
			"Sorry, my database seems to be down. Come later!",
			errors.Wrap(err, "cannot insert form"),
		)
	}

	if err := h.DB.UpdateCurrentFormID(&model.User{
		ID: user.ID,
		CurrentFormID: sql.NullInt64{
			Int64: int64(form.ID),
			Valid: true,
		},
	}); err != nil {
		return tool.NewHRError(
			"Sorry, my database seems to be down. Come later!",
			errors.Wrap(err, "cannot update current_form_id"),
		)
	}

	if err := h.DB.UpdateState(&model.User{
		TelegramID: update.Message.From.ID,
		State: sql.NullInt64{
			Int64: 1,
			Valid: true,
		},
	}); err != nil {
		return tool.NewHRError(
			"Sorry, my database seems to be down. Come later!",
			errors.Wrap(err, "cannot update state"),
		)
	}

	resp, err := http.Get(form.CaptchaLink)
	if err != nil {
		return tool.NewHRError(
			"Sorry, seems like balance checker service is down. Nothing to deal with it. Come later!",
			errors.Wrap(err, "cannot download captcha"),
		)
	}
	defer resp.Body.Close()

	msg := tgbotapi.NewPhotoUpload(update.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "captcha.jpg",
		Reader: resp.Body,
		Size:   -1,
	})
	msg.Caption = "Enter captcha:"

	// I don't know fucking why, but without timer we are getting "incorrect code" error
	select {
	case <-time.After(time.Second * 3):
		if _, err := h.Telegram.Send(msg); err != nil {
			return errors.Wrap(err, "cannot send message")
		}
	}

	return nil
}

func (h *Handler) RemoveCardAttention(update tgbotapi.Update) error {
	if len(update.Message.Text) < strings.LastIndex(update.Message.Text, "_")+2 {
		return tool.NewHRError(
			"Incorrect command arguments.",
			errors.New("there is no card number"),
		)
	}

	cardID, err := strconv.Atoi(update.Message.Text[strings.LastIndex(update.Message.Text, "_")+1:])
	if err != nil {
		return tool.NewHRError(
			"Incorrect command arguments.",
			errors.Wrap(err, "cannot convert card number into int"),
		)
	}

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf("Please, confirm card deletion: /rm_confirm_%d", cardID),
	)

	if _, err := h.Telegram.Send(msg); err != nil {
		return errors.Wrap(err, "cannot send message")
	}

	return nil
}

func (h *Handler) RemoveCard(update tgbotapi.Update) error {
	user, err := h.DB.GetUser(&model.User{
		TelegramID: update.Message.From.ID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return tool.NewHRError(
				"Hm. Seems like I cannot find record about you. Try again by clicking /start!",
				errors.Wrap(err, "user somehow didn't click /start"),
			)
		} else {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot get user"),
			)
		}
	}

	if len(update.Message.Text) < strings.LastIndex(update.Message.Text, "_")+2 {
		return tool.NewHRError(
			"Incorrect command arguments.",
			errors.New("there is no card number"),
		)
	}

	cardID, err := strconv.Atoi(update.Message.Text[strings.LastIndex(update.Message.Text, "_")+1:])
	if err != nil {
		return tool.NewHRError(
			"Incorrect command arguments.",
			errors.Wrap(err, "cannot convert card number into int"),
		)
	}

	if err := h.DB.DeleteCard(&model.Card{
		ID:     cardID,
		UserID: user.ID,
	}); err != nil {
		return tool.NewHRError(
			"Sorry, my database seems to be down. Come later!",
			errors.Wrap(err, "cannot delete card"),
		)
	}

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		"Card deleted.",
	)

	if _, err := h.Telegram.Send(msg); err != nil {
		return errors.Wrap(err, "cannot send message")
	}

	return nil
}

func (h *Handler) Cancel(update tgbotapi.Update) error {
	user, err := h.DB.GetUser(&model.User{
		TelegramID: update.Message.From.ID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return tool.NewHRError(
				"Hm. Seems like I cannot find record about you. Try again by clicking /start!",
				errors.Wrap(err, "user somehow didn't click /start"),
			)
		} else {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot get user"),
			)
		}
	}

	if err := h.DB.UpdateState(&model.User{
		ID: user.ID,
		State: sql.NullInt64{
			Int64: 0,
			Valid: true,
		},
	}); err != nil {
		return tool.NewHRError(
			"Sorry, my database seems to be down. Come later!",
			errors.Wrap(err, "cannot update state"),
		)
	}

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		"Operation canceled.",
	)

	if _, err := h.Telegram.Send(msg); err != nil {
		return errors.Wrap(err, "cannot send message")
	}

	return nil
}

func (h *Handler) Default(update tgbotapi.Update) error {
	user, err := h.DB.GetUser(&model.User{
		TelegramID: update.Message.From.ID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return tool.NewHRError(
				"Hm. Seems like I cannot find record about you. Try again by clicking /start!",
				errors.Wrap(err, "user somehow didn't click /start"),
			)
		} else {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot get user"),
			)
		}
	}

	// Skip user without state
	if !user.State.Valid {
		return nil
	}

	switch user.State.Int64 {
	// We are waiting card id
	case 0:
		cardNumber, err := strconv.Atoi(update.Message.Text)
		if err != nil {
			return tool.NewHRError(
				"Enter number please.",
				errors.Wrap(err, "cannot convert into int"),
			)
		}

		card, err := h.DB.InsertCard(&model.Card{
			UserID: user.ID,
			Number: cardNumber,
		})
		if err != nil {
			if err, ok := err.(pq.PGError); ok {
				if err.Get('C') == "23505" {
					card, _ = h.DB.GetCard(&model.Card{
						Number: cardNumber,
						UserID: user.ID,
					})
				} else {
					return tool.NewHRError(
						"Sorry, my database seems to be down. Come later!",
						errors.Wrap(err, "cannot insert card"),
					)
				}
			} else {
				return tool.NewHRError(
					"Sorry, my database seems to be down. Come later!",
					errors.Wrap(err, "cannot insert card"),
				)
			}
		}

		if err := h.DB.UpdateCurrentCardID(&model.User{
			ID: user.ID,
			CurrentCardID: sql.NullInt64{
				Int64: int64(card.ID),
				Valid: true,
			},
		}); err != nil {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot update current_card_id"),
			)
		}

		form, err := tool.GetForm()
		if err != nil {
			return tool.NewHRError(
				"Sorry, seems like balance checker service is down. Nothing to deal with it. Come later!",
				errors.Wrap(err, "cannot get form"),
			)
		}

		// Don't forget to set card_id before form insertion
		form.CardID = card.ID
		form, err = h.DB.InsertForm(form)
		if err != nil {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot insert form"),
			)
		}

		if err := h.DB.UpdateCurrentFormID(&model.User{
			ID: user.ID,
			CurrentFormID: sql.NullInt64{
				Int64: int64(form.ID),
				Valid: true,
			},
		}); err != nil {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot update current_card_id"),
			)
		}

		if err := h.DB.UpdateState(&model.User{
			TelegramID: update.Message.From.ID,
			State: sql.NullInt64{
				Int64: 1,
				Valid: true,
			},
		}); err != nil {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot update state"),
			)
		}

		resp, err := http.Get(form.CaptchaLink)
		if err != nil {
			return tool.NewHRError(
				"Sorry, seems like balance checker service is down. Nothing to deal with it. Come later!",
				errors.Wrap(err, "cannot download captcha"),
			)
		}
		defer resp.Body.Close()

		msg := tgbotapi.NewPhotoUpload(update.Message.Chat.ID, tgbotapi.FileReader{
			Name:   "captcha.jpg",
			Reader: resp.Body,
			Size:   -1,
		})
		msg.Caption = "Enter captcha:"

		// I don't know fucking why, but without timer we are getting "incorrect code" error
		select {
		case <-time.After(time.Second * 3):
			if _, err := h.Telegram.Send(msg); err != nil {
				return errors.Wrap(err, "cannot send message")
			}
		}

		return nil
	case 1:
		if !user.CurrentCardID.Valid {
			return errors.Wrap(err, "invalid state")
		}

		card, err := h.DB.GetCard(&model.Card{
			ID:     int(user.CurrentCardID.Int64),
			UserID: user.ID,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				return tool.NewHRError(
					"Hm. Seems like I cannot find record about you. Please, try again.",
					errors.Wrap(err, "cannot find card"),
				)
			} else {
				return tool.NewHRError(
					"Sorry, my database seems to be down. Come later!",
					errors.Wrap(err, "cannot get card"),
				)
			}
		}

		if !user.CurrentFormID.Valid {
			return errors.Wrap(err, "invalid state")
		}

		form, err := h.DB.GetForm(&model.Form{
			ID:     int(user.CurrentFormID.Int64),
			CardID: card.ID,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				return tool.NewHRError(
					"Hm. Seems like I cannot find record about you. Please, try again.",
					errors.Wrap(err, "cannot find form"),
				)
			} else {
				return tool.NewHRError(
					"Sorry, my database seems to be down. Come later!",
					errors.Wrap(err, "cannot get form"),
				)
			}
		}

		if err := h.DB.UpdateState(&model.User{
			TelegramID: update.Message.From.ID,
			State: sql.NullInt64{
				Valid: false,
			},
		}); err != nil {
			return tool.NewHRError(
				"Sorry, my database seems to be down. Come later!",
				errors.Wrap(err, "cannot update state"),
			)
		}

		code, err := strconv.Atoi(update.Message.Text)
		if err != nil {
			return tool.NewHRError(
				"Enter valid code.",
				errors.Wrap(err, "cannot convert into int"),
			)
		}

		text, err := tool.PostForm(form.ViewState, form.EventValidation, card.Number, code)
		if err != nil {
			if errors.Cause(err) == tool.IncorrectCode {
				return tool.NewHRError(
					"Captcha code is incorrect! Please, start operation again.",
					errors.Wrap(err, "cannot get form result"),
				)
			}

			return tool.NewHRError(
				"Sorry, internal bot error happened. Please, try again or come later.",
				errors.Wrap(err, "cannot get form result"),
			)
		}

		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			text,
		)
		msg.ParseMode = "HTML"

		if _, err := h.Telegram.Send(msg); err != nil {
			return errors.Wrap(err, "cannot send message")
		}

		return nil
	default:
		return nil
	}
}
