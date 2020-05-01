package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Processor implementa i metodi per processare comandi e messaggi semplici
type Processor interface {
	Help() string
	Version() string

	// Restituiscono true quando il comando o messaggio è stato effettivamente processato
	ProcessUpdate(update tgbotapi.Update) (bool, error)
	ProcessCommand(handler MessageHandler, command string, params []string) (bool, error)
	ProcessMessage(handler MessageHandler, text string) (bool, error)
}

func (bot *Bot) ProcessUpdate(update tgbotapi.Update) (bool, error) {

	if bot.Debug {
		log.Printf("(new update) %+v %+v\n", update.Message, update.EditedMessage)
	}

	var replyUserID int
	var replyUsername string

	allowed, canProcessCommands := bot.allowedUpdate(&update)
	if !allowed {
		return true, nil
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

	if message == nil {
		return false, nil
	}

	if message.ReplyToMessage != nil {
		replyUserID = message.ReplyToMessage.From.ID
		replyUsername = message.ReplyToMessage.From.UserName
	}

	handler := MessageHandler{
		UserID:        message.From.ID,
		Username:      message.From.UserName,
		ChatID:        message.Chat.ID,
		IsPrivate:     message.Chat.IsPrivate(),
		MessageID:     message.MessageID,
		ReplyUserID:   replyUserID,
		ReplyUsername: replyUsername,
	}
	if edited {
		handler.EditMessageID = bot.sentMessages.lookupSenderSent[message.MessageID]
	}

	if canProcessCommands {
		processed, err := bot.processMessageAsCommand(message, handler)
		if err != nil {
			return true, err
		}
		if processed {
			return true, nil
		}
	}

	if bot.config.ProcessGroupMessages && !bot.silenceOn {
		// Delega i messaggi semplici ai processori.
		// il primo che processa interrompe la coda.
		for _, p := range bot.processors {
			processed, err := p.ProcessMessage(handler, message.Text)
			if err != nil {
				return true, err
			}
			if processed {
				break
			}
		}
	}

	return true, nil
}

func (bot *Bot) processMessageAsCommand(message *tgbotapi.Message, handler MessageHandler) (bool, error) {
	var command string
	var params []string
	var ok bool

	command, params, ok = bot.parseCommand(message)

	if !ok {
		return false, nil
	}

	if bot.Debug {
		log.Printf("(stack) Command from %s/%s: %s %v\n", message.Chat.Title, message.From.UserName, command, params)
	}

	u, ok := bot.getUserByID(message.From.ID)
	if ok {
		handler.Group = u.Group

		// risincronizza se necessario i dati dell'utente
		if u.Username != message.From.UserName {
			bot.updateUsername(u, message.From.UserName)
		}
		if handler.IsPrivate &&
			(u.PrivateChatID != message.Chat.ID) {
			bot.updateUserPrivateChatID(u, message.Chat.ID)
		}
	}

	// Delega i comandi ai processori.
	// il primo che processa interrompe la coda.
	for _, p := range bot.processors {
		processed, err := p.ProcessCommand(handler, command, params)
		if err != nil {
			return true, err
		}
		if processed {
			break
		}
	}

	return true, nil
}
