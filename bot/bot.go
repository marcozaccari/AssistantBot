package bot

// TODO: config ereditabile
// TODO: permessi utenti (ora tutti comandano tutto - argh)

import (
	"errors"
	"log"
	"sync"

	"modulo.srl/AssistantBot/settings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Bot - incapsula lo stack di Telegram
type Bot struct {
	Debug bool

	username string

	configCtrl *settings.Settings
	config     configData
	configLock sync.Mutex // serializza le scritture in config

	lookupUsers usersLookupMap

	tgbot *tgbotapi.BotAPI
}

type CommandsProcessor interface {
	ProcessCommand(handler CommandHandler, command string, params []string) error
}

// Init - inizializza il bot.
// configFilename: assoluto o relativo al path dell'eseguibile.
func (bot *Bot) Init(configFilename string, verbose bool, debug bool) error {
	var err error

	bot.Debug = debug

	if verbose {
		log.Printf("Initializing...")
	}

	err = bot.initSettings(configFilename, true, verbose)
	if err != nil {
		return err
	}

	if verbose {
		log.Printf("Init stack (telegram-bot-api)")
	}
	if debug {
		log.Printf("SecureToken \"%s\"", bot.config.SecureToken)
	}

	bot.tgbot, err = tgbotapi.NewBotAPI(bot.config.SecureToken)
	if err != nil {
		return errors.New("(stack) " + err.Error())
	}

	//bot.tgbot.Debug = true

	if verbose {
		bot.username = bot.tgbot.Self.UserName
		log.Printf("(stack) Bot username \"%s\"", bot.tgbot.Self.UserName)
	}

	bot.initUsers(verbose)

	return nil
}

// - loadConfig: carica anche il file
func (bot *Bot) initSettings(configFilename string, loadConfig bool, verbose bool) error {
	var err error

	bot.config = configData{}
	bot.configCtrl, err = settings.New(configFilename, &bot.config, verbose)
	if err != nil {
		return err
	}

	if loadConfig {
		err = bot.configCtrl.LoadSettings()
		if err != nil {
			return err
		}
	}

	return nil
}

// Do - processa in modo bloccante le updates di Telegram.
func (bot *Bot) Do(processor CommandsProcessor) error {
	var offset int

	if !bot.config.RecoverOldUpdates {
		offset = -1
	}

	u := tgbotapi.NewUpdate(offset)
	u.Timeout = 300

	updates, err := bot.tgbot.GetUpdatesChan(u)
	if err != nil {
		return errors.New("(stack) " + err.Error())
	}

	log.Println("Listening for updates...")

	// bloccante
	for update := range updates {
		err = bot.processUpdate(update, processor)
		if err != nil {
			return err
		}
	}

	return nil
}

func (bot *Bot) sendMessage(chatID int64, messageID int, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = messageID

	//	msg.ParseMode = "MarkdownV2"
	msg.ParseMode = "HTML"

	bot.tgbot.Send(msg)
}

func (bot *Bot) sendCommandResponse(handler CommandHandler, text string, toPrivateChat bool,
	quote bool, replace bool) {

	var chatID int64

	if toPrivateChat && !handler.isPrivate {
		u, ok := bot.getUserByID(handler.userID)
		if !ok {
			log.Println("Cannot send to private chat", handler, text)
			return
		}
		chatID = u.PrivateChatID
	} else {
		chatID = handler.chatID
	}

	var messageID int
	if quote && (chatID == handler.chatID) && !replace {
		messageID = handler.messageID
	}

	bot.sendMessage(chatID, messageID, text)

	if replace {
		cfg := tgbotapi.DeleteMessageConfig{
			ChatID:    handler.chatID,
			MessageID: handler.messageID,
		}

		bot.tgbot.DeleteMessage(cfg)
	}
}

func (bot *Bot) SendCommandResponseToPrivate(handler CommandHandler, text string) {
	bot.sendCommandResponse(handler, text, true, !handler.isPrivate, false)

	if !handler.isPrivate {
		bot.sendCommandResponse(handler, "pvt", false, true, false)
	}
}

func (bot *Bot) SendCommandResponse(handler CommandHandler, text string, replace bool) {
	bot.sendCommandResponse(handler, text, false, !handler.isPrivate, replace)
}

// CreateBlankConfig crea un nuovo file di configurazione vuoto
func CreateBlankConfig(configFilename string) error {
	log.Println("Create a blank config file...")

	tempBot := Bot{}

	err := tempBot.initSettings(configFilename, false, true)
	if err != nil {
		return err
	}

	err = tempBot.configCtrl.SaveSettings()
	if err != nil {
		return err
	}

	return nil
}
