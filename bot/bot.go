package bot

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"modulo.srl/AssistantBot/settings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const version = "1.0.0"

// Bot - incapsula lo stack di Telegram
type Bot struct {
	Debug   bool
	Verbose bool

	username string
	userID   int

	configs      map[string]interface{}
	configCtrl   *settings.Settings
	configLoaded bool

	config     configData
	configLock sync.Mutex // serializza le scritture in config

	lookupUsers usersLookupMap

	sentMessages sentMessagesLookups

	processors []Processor

	silenceOn    bool
	timerSilence *time.Timer

	tgbot *tgbotapi.BotAPI
}

// Processor implementa i metodi per processare comandi e messaggi semplici
type Processor interface {
	Help() string
	// Restituiscono true quando il comando o messaggio è stato effettivamente processato
	ProcessCommand(handler MessageHandler, command string, params []string) (bool, error)
	ProcessMessage(handler MessageHandler, text string) (bool, error)
}

func (bot *Bot) initStack() error {
	if bot.Verbose {
		log.Printf("Init stack (telegram-bot-api)")
	}
	if bot.Debug {
		log.Printf("SecureToken \"%s\"", bot.config.SecureToken)
	}

	var err error
	bot.tgbot, err = tgbotapi.NewBotAPI(bot.config.SecureToken)
	if err != nil {
		return errors.New("(stack) " + err.Error())
	}

	//bot.tgbot.Debug = true

	if bot.Verbose {
		bot.username = bot.tgbot.Self.UserName
		bot.userID = bot.tgbot.Self.ID
		log.Printf("(stack) Bot username \"%s\"", bot.tgbot.Self.UserName)
	}

	bot.initUsers()
	bot.initMessages()

	return nil
}

func (bot *Bot) init(configFilename string) error {
	var err error

	bot.configs = make(map[string]interface{})
	bot.processors = []Processor{}

	bot.config = configData{}
	bot.RegisterProcessor("bot", bot, &bot.config)

	bot.configCtrl, err = settings.New(configFilename, bot.configs, bot.Verbose)
	if err != nil {
		return err
	}

	return nil
}

// RegisterProcessor aggiunge un processor al bot, estendendone le funzionalità
func (bot *Bot) RegisterProcessor(name string, processor Processor, configData interface{}) {
	bot.processors = append(bot.processors, processor)
	bot.RegisterConfig(strings.Title(name), configData)
}

// Do - processa in modo bloccante le updates di Telegram.
func (bot *Bot) Do() error {
	var err error
	var offset int

	if !bot.configLoaded {
		err = bot.LoadConfig()
		if err != nil {
			return err
		}
	}

	err = bot.initStack()
	if err != nil {
		return err
	}

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
		err = bot.processUpdate(update)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewBot(configFilename string, verbose bool, debug bool) *Bot {
	bot := Bot{}

	bot.Debug = debug
	bot.Verbose = verbose

	bot.init(configFilename)

	return &bot
}
