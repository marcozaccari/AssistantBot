package bot

import (
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func (bot *Bot) processUpdate(update tgbotapi.Update, cProcessor CommandsProcessor) error {

	if bot.Debug {
		log.Printf("(new update) %+v\n", update.Message)
	}

	var command string
	var params []string
	var chatID int64
	var isPrivateChat bool
	var replyUserID int
	var replyUsername string

	if update.Message != nil {
		command, params = bot.parseCommand(update.Message)
		if command == "" {
			return nil
		}
	}

	if !bot.AllowedUpdate(&update, command) {
		return nil
	}

	// parse update

	if update.Message != nil {
		chatID = update.Message.Chat.ID
		isPrivateChat = update.Message.Chat.IsPrivate()

		if update.Message.ReplyToMessage != nil {
			replyUserID = update.Message.ReplyToMessage.From.ID
			replyUsername = update.Message.ReplyToMessage.From.UserName
		}

		if bot.Debug {
			log.Printf("(stack) Command from %s/%s: %s %v\n", update.Message.Chat.Title, update.Message.From.UserName, command, params)
		}

		u, ok := bot.getUserByID(update.Message.From.ID)
		if ok {
			// risincronizza se necessario i dati dell'utente
			if u.Username != update.Message.From.UserName {
				bot.updateUsername(u, update.Message.From.UserName)
			}
			if isPrivateChat &&
				(u.PrivateChatID != chatID) {
				bot.updateUserPrivateChatID(u, chatID)
			}
		}

		handler := CommandHandler{
			userID:        update.Message.From.ID,
			chatID:        chatID,
			isPrivate:     isPrivateChat,
			messageID:     update.Message.MessageID,
			replyUserID:   replyUserID,
			replyUsername: replyUsername,
		}

		// command is preparsed above
		switch command {

		case superCommandOwner:
			// reset owner
			if len(params) > 0 {
				if params[0] == bot.config.SecureToken {
					bot.resetOwner(update.Message.From.ID, update.Message.From.UserName, chatID)

					text := "You are the owner of this bot, now"
					bot.sendMessage(chatID, update.Message.MessageID, text)
				}
			}

		case commandUser:
			err := bot.processUserCommand(handler, params)
			if err != nil {
				return err
			}

		default:
			if cProcessor != nil {
				err := cProcessor.ProcessCommand(handler, command, params)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Parsa un messaggio ottenendo comando e parametri.
// I comandi vengono recepiti nei seguenti modi:
// - standard: /comando parametri oppure /comando@bot parametri
// - citazione: @bot comando parametri
// - messaggio privato: comando parametri
// - risposta a un messaggio del bot: comando parametri
// - parola d'ordine: config.CommandWord comando parametri
func (bot *Bot) parseCommand(message *tgbotapi.Message) (command string, params []string) {
	if cmd := message.Command(); cmd != "" {
		// /comando standard
		command = cmd
		params = strings.Fields(message.CommandArguments())
		return
	}

	fields := strings.Fields(message.Text)
	var paramsIdx int

	if fields[0] == "@"+bot.username || fields[0] == bot.config.CommandWord {
		if len(fields) > 1 {
			command = fields[1]
			paramsIdx = 2
		}
	} else {
		if !message.Chat.IsPrivate() {
			return
		}

		command = fields[0]
		paramsIdx = 1
	}

	if len(fields) <= paramsIdx {
		params = []string{}
	} else {
		params = make([]string, len(fields)-paramsIdx)
		copy(params, fields[paramsIdx:])
	}

	return
}
