package bot

// Funzioni di filtraggio messaggi in input

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Restituisce true se il messaggio può essere processato
func (bot *Bot) AllowedUpdate(update *tgbotapi.Update, command string) bool {
	if update.Message != nil {
		// Message

		if command == superCommandOwner {
			// i super comandi possono essere ricevuti da sconosciuti, ma solo in chat private
			if !update.Message.Chat.IsPrivate() {
				if bot.Debug {
					log.Println("(firewall) super command not allowed in public chats")
				}
				return false
			}

			return true
		}

		_, ok := bot.getUserByID(update.Message.From.ID)

		if !ok {
			if bot.Debug {
				log.Println("(firewall) From ID", update.Message.From.ID, "not in allowed IDs")
			}
			return false
		}
	}

	return true
}
