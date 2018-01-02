package bot

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"gopkg.in/telegram-bot-api.v4"
)

type SpamReport struct {
	Pk              int `storm:"id,increment"`
	UserID          int
	MessageID       int
	ReportMessageID int
	CreatedAt       time.Time
}

func isReplyAnReport(msg string, botName string) bool {
	msg = strings.ToLower(msg)
	// remove bot name from message, i.e /spam@spam_kill_robot
	msg = strings.Replace(msg, "@"+botName, "", -1)
	switch msg {
	case
		"spam",
		"/spam",
		"спам",
		"/спам",
		"report",
		"/report":
		return true
	}
	return false
}

func Run(dbPath, token string, votes int, debug bool) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	bot.Debug = debug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	db, err := storm.Open(dbPath)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	defer db.Close()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		msgText := update.Message.Text

		if len(msgText) == 0 {
			log.Printf("Empty text: %d", update.Message.MessageID)
			continue
		}

		if update.Message.ReplyToMessage == nil {
			log.Printf("Not reply: %s", update.Message.Text)
			continue
		}

		if isReplyAnReport(msgText, bot.Self.UserName) == false {
			log.Printf("Not report: %d", update.Message.MessageID)
			continue
		}

		msgFromID := update.Message.From.ID
		msgID := update.Message.ReplyToMessage.MessageID

		var numReports int
		numReports, err = db.Select(q.Eq("MessageID", msgID)).Count(new(SpamReport))
		log.Printf("Number of reports: %d", numReports)

		if votes == numReports+1 {
			// remove the spam post from the chat
			dm := tgbotapi.DeleteMessageConfig{ChatID: update.Message.Chat.ID, MessageID: msgID}
			_, err := bot.DeleteMessage(dm)

			if err == nil {
				// fetch all reports and delete them
				var reports []SpamReport
				err = db.Select(q.Eq("MessageID", msgID)).Find(&reports)
				if err == nil {
					for _, report := range reports {
						// remove the post from the chat
						dm := tgbotapi.DeleteMessageConfig{
							ChatID:    update.Message.Chat.ID,
							MessageID: report.ReportMessageID,
						}
						_, err := bot.DeleteMessage(dm)
						if err != nil {
							log.Printf("An error with deleting the message: %s", err)
						}
					}
				}

				// Remove posts from the database
				err = db.Select(q.Eq("MessageID", msgID)).Delete(new(SpamReport))
				if err != nil {
					log.Printf("An error with deleting messages from the database: %s", err)
				}
			}
		}

		var sr SpamReport
		err := db.Select(q.Eq("UserID", msgFromID), q.Eq("MessageID", msgID)).First(&sr)
		if err == storm.ErrNotFound {
			// save report
			sr = SpamReport{
				UserID:          msgFromID,
				MessageID:       msgID,
				ReportMessageID: update.Message.MessageID,
				CreatedAt:       time.Now(),
			}

			err = db.Save(&sr)
			if err != nil {
				log.Printf("An error with saving a report: %s", err)
			} else {
				fMsg := fmt.Sprintf("ОК! Нужно еще голосов: %d", votes-numReports-1)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fMsg)
				msg.ReplyToMessageID = update.Message.MessageID

				bot.Send(msg)
			}
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы уже отдали свой голос!")
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}
	}
}
