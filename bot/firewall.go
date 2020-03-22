package bot

// Funzioni di filtraggio messaggi in input

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Restituisce true se il messaggio può essere processato
func (bot *Bot) allowedUpdate(update *tgbotapi.Update) (allowed bool, canProcessCommands bool) {
	var message *tgbotapi.Message

	if update.Message != nil {
		message = update.Message
	} else if update.EditedMessage != nil {
		message = update.EditedMessage
	}

	if message != nil {
		// Message

		if message.From.IsBot {
			// i messaggi dei bot vengono scartati
			return false, false
		}

		isPrivateChat := message.Chat.IsPrivate()

		if isPrivateChat && (message.Command() == superCommandOwner) {
			// i super comandi possono essere ricevuti da sconosciuti, ma solo in chat private
			allowed = true
			canProcessCommands = true
			return
		}

		_, ok := bot.getUserByID(message.From.ID)
		if ok {
			allowed = true
			canProcessCommands = true
			return
		}

		if bot.config.ProcessGroupMessages && !isPrivateChat {
			allowed = true
			canProcessCommands = false
			return
		}

		if bot.Debug {
			log.Println("(firewall) From ID", message.From.ID, "not in allowed IDs")
		}
	}

	return
}
