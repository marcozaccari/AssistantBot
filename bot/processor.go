package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (bot *Bot) processUpdate(update tgbotapi.Update) error {

	if bot.Debug {
		log.Printf("(new update) %+v %+v\n", update.Message, update.EditedMessage)
	}

	var command string
	var params []string
	var chatID int64
	var isPrivateChat bool
	var replyUserID int
	var replyUsername string

	allowed, canProcessCommands := bot.allowedUpdate(&update)
	if !allowed {
		return nil
	}

	// parse update
	var message *tgbotapi.Message
	var edited bool

	if update.Message != nil {
		message = update.Message
	} else if update.EditedMessage != nil {
		message = update.EditedMessage
		edited = true
	}

	if message != nil {
		chatID = message.Chat.ID
		isPrivateChat = message.Chat.IsPrivate()

		if message.ReplyToMessage != nil {
			replyUserID = message.ReplyToMessage.From.ID
			replyUsername = message.ReplyToMessage.From.UserName
		}

		handler := MessageHandler{
			userID:        message.From.ID,
			username:      message.From.UserName,
			chatID:        chatID,
			isPrivate:     isPrivateChat,
			messageID:     message.MessageID,
			replyUserID:   replyUserID,
			replyUsername: replyUsername,
		}
		if edited {
			handler.editMessageID = bot.sentMessages.lookupSenderSent[message.MessageID]
		}

		if canProcessCommands {
			var ok bool
			command, params, ok = bot.parseCommand(message)
			if ok {

				if bot.Debug {
					log.Printf("(stack) Command from %s/%s: %s %v\n", message.Chat.Title, message.From.UserName, command, params)
				}

				u, ok := bot.getUserByID(message.From.ID)
				if ok {
					handler.group = u.Group

					// risincronizza se necessario i dati dell'utente
					if u.Username != message.From.UserName {
						bot.updateUsername(u, message.From.UserName)
					}
					if isPrivateChat &&
						(u.PrivateChatID != chatID) {
						bot.updateUserPrivateChatID(u, chatID)
					}
				}

				for _, p := range bot.processors {
					processed, err := p.ProcessCommand(handler, command, params)
					if err != nil {
						return err
					}
					if processed {
						break
					}
				}

				return nil
			}
		}

		// controlla se è il caso di processare i messaggi semplici
		if bot.config.ProcessGroupMessages && !bot.silenceOn {
			for _, p := range bot.processors {
				processed, err := p.ProcessMessage(handler, message.Text)
				if err != nil {
					return err
				}
				if processed {
					break
				}
			}
		}
	}

	return nil
}
