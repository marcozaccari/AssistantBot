package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Resetta l'owner del bot; pretende secureToken come parametro
const superCommandOwner = "owner"

// gestisce i permessi
const commandUser = "user"

const commandPing = "ping"

const commandSilence = "silence"

func (bot *Bot) Help() string {
	return "/owner  Reset the owner\n" +
		"/user  Users commands\n" +
		"/silence [off]  Stop (or restart) messages autoparsing\n" +
		"/ping  Test the bot\n"
}

// Parsa un messaggio ottenendo comando e parametri.
// I comandi vengono recepiti nei seguenti modi:
// - standard: /comando parametri oppure /comando@bot parametri
// - messaggio privato: comando parametri
// - citazione: @bot comando parametri
// - parola d'ordine: config.CommandWord comando parametri
// - risposta a un messaggio del bot: comando parametri
func (bot *Bot) parseCommand(message *tgbotapi.Message) (command string, params []string, ok bool) {
	if cmd := message.Command(); cmd != "" {
		// /comando standard
		command = cmd
		params = strings.Fields(message.CommandArguments())
		ok = true
		return
	}

	text := message.Text

	// controlla il match della prima parola
	matchFirstWord := func(firstWord string) (bool, string) {
		if len(firstWord) > 1 {
			firstWord += " "
		}

		wlen := len(firstWord)

		if len(text) <= wlen {
			return false, ""
		}

		if text[0:wlen] == firstWord {
			return true, text[wlen:]
		}

		return false, ""
	}

	var fields []string
	ok = false
	isCommand := false

	if message.Chat.IsPrivate() {
		isCommand = true
	} else {
		// group chat
		if (message.ReplyToMessage != nil) && (message.ReplyToMessage.From.ID == bot.userID) {
			isCommand = true
		} // else evitato per permettere @bot o parolaOrdine anche rispondendo al bot

		if b, newText := matchFirstWord("@" + bot.username); b {
			isCommand = true
			text = newText
		} else if bot.config.CommandWord != "" {
			if b, newText := matchFirstWord(bot.config.CommandWord); b {
				isCommand = true
				text = newText
			}
		}
	}

	if isCommand {
		fields = strings.Fields(text)
		if len(fields) == 0 {
			return
		}

		command = fields[0]
		ok = true

		if len(fields) == 1 {
			// senza parametri
			params = []string{}
			return
		}

		params = fields[1:]
	}

	return
}

func (bot *Bot) ProcessCommand(handler MessageHandler, command string, params []string) (bool, error) {
	switch command {

	case "start":
		if len(params) > 0 {
			// In presenza di parametri evita di processare, poichè potrebbe trattarsi di una InlineQuery
			// probabilmente gestita da un altro processore applicativo.
			return false, nil
		}
		fallthrough

	case "help":
		var help string
		for _, p := range bot.processors {
			help += p.Help() + "\n"
		}
		opt := bot.NewMessageResponseOpt()
		bot.SendMessageResponseToPrivate(handler, help, opt)

	case superCommandOwner:
		// reset owner
		if len(params) > 0 {
			if params[0] == bot.config.SecureToken {
				bot.resetOwner(handler.UserID, handler.Username, handler.ChatID)

				text := "You are the owner of this bot, now"
				opt := bot.NewMessageResponseOpt()
				bot.SendMessageResponse(handler, text, opt)
			}
		}
		return true, nil

	case commandPing:
		var text string
		for i, p := range bot.processors {
			text += bot.processorsNames[i] + " <code>" + p.Version() + "</code>\n"
		}

		opt := bot.NewMessageResponseOpt()
		bot.SendMessageResponse(handler, text, opt)

	case commandSilence:
		bot.processSilenceCommand(handler, params)

	case commandUser:
		err := bot.processUserCommand(handler, params)
		if err != nil {
			return true, err
		}
		return true, nil
	}

	return false, nil
}

func (bot *Bot) processSilenceCommand(handler MessageHandler, params []string) {
	if len(params) > 0 && params[0] == "off" {
		bot.silenceOn = false
		if bot.timerSilence != nil {
			bot.timerSilence.Stop()
		}

		text := "Silence mode off"
		opt := bot.NewMessageResponseOpt()
		bot.SendMessageResponse(handler, text, opt)
		return
	}

	mins := bot.config.SilenceTimeoutMins
	if mins == 0 {
		mins = 30
	}
	timeout := time.Minute * time.Duration(mins)
	bot.silenceOn = true

	if bot.timerSilence != nil {
		bot.timerSilence.Reset(timeout)
	} else {
		endSilence := func() {
			bot.silenceOn = false
			if bot.Verbose {
				log.Println("Silence mode off")
			}
		}

		bot.timerSilence = time.AfterFunc(timeout, endSilence)
	}

	text := fmt.Sprintf("Silenced for <code>%v</code> minutes", mins)
	opt := bot.NewMessageResponseOpt()
	bot.SendMessageResponse(handler, text, opt)
}
