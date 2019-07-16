package main

import (
	"flag"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"strings"
	"tc-bot/config"
	"tc-bot/database"
	"tc-bot/handler"
	"tc-bot/tool"
)

func main() {
	var path string

	flag.StringVar(
		&path,
		"config",
		"",
		"enter path to config file",
	)

	// Parse at first startup
	flag.Parse()

	// Init logger
	logger := logrus.New()

	// Get config
	conf, err := config.NewConfig(path)
	if err != nil {
		logger.WithError(err).Fatal("incorrect path or config itself")
	}

	// Set log level from config
	lvl, err := logrus.ParseLevel(conf.LogLevel)
	if err != nil {
		logger.WithError(err).Fatal("cannot parse log level")
	}

	logger.SetLevel(lvl)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	// Connect database
	db, err := sqlx.Connect("postgres", conf.DB.ConnectionString())
	if err != nil {
		logger.WithError(err).Fatal("cannot connect to database")
	}

	bot, err := tgbotapi.NewBotAPI(conf.Telegram.Token)
	if err != nil {
		fmt.Println("Telegram bot cannot be initialized! See, error:")
		panic(err)
	}

	fmt.Printf("Authorized on account @%s\n", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	// Graceful shutdown
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, os.Kill)

	go func() {
		<-s
		updates.Clear()
		os.Exit(1)
	}()

	h := handler.NewHandler(database.NewInstance(db), bot, logger, conf)
	for update := range updates {
		go func(u tgbotapi.Update) {
			if u.Message == nil { // ignore any non-Message Updates
				return
			}

			handleError := func(err error) {
				// Log error
				h.Logger.Error(err)

				// Send human readable representation of error to user to let him know
				if hrerr, ok := err.(*tool.HRError); ok {
					msg := tgbotapi.NewMessage(u.Message.Chat.ID, hrerr.Human())
					_, err := h.Telegram.Send(msg)
					if err != nil {
						h.Logger.Error(errors.Wrap(err, "cannot send message with human readable error"))
					}
				} else {
					// ... do nothing? Unreadable error useless for people
				}
			}

			switch u.Message.Command() {
			case "start":
				if err := h.Start(u); err != nil {
					handleError(err)
				}
				return
			case "addcard":
				if err := h.AddCard(u); err != nil {
					handleError(err)
				}
				return
			case "cards":
				if err := h.Cards(u); err != nil {
					handleError(err)
				}
				return
			case "cancel":
				if err := h.Cancel(u); err != nil {
					handleError(err)
				}
				return
			default:
				if strings.HasPrefix(u.Message.Text, "/balance_") {
					if err := h.Balance(u); err != nil {
						handleError(err)
					}
					return
				}
				if strings.HasPrefix(u.Message.Text, "/remove_") {
					if err := h.RemoveCardAttention(u); err != nil {
						handleError(err)
					}
					return
				}
				if strings.HasPrefix(u.Message.Text, "/rm_confirm_") {
					if err := h.RemoveCard(u); err != nil {
						handleError(err)
					}
					return
				}
				if err := h.Default(u); err != nil {
					handleError(err)
				}
				return
			}
		}(update)
	}
}
