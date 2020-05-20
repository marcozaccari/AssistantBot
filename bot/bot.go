package bot

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/marcozaccari/AssistantBot/settings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const version = "1.0.2"

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

	processors      []Processor
	processorsNames []string

	silenceOn    bool
	timerSilence *time.Timer

	Tgbot *tgbotapi.BotAPI
}

func (bot *Bot) initStack() error {
	if bot.Verbose {
		log.Printf("Init stack (telegram-bot-api)")
	}
	if bot.Debug {
		log.Printf("SecureToken \"%s\"", bot.config.SecureToken)
	}

	var err error
	bot.Tgbot, err = tgbotapi.NewBotAPI(bot.config.SecureToken)
	if err != nil {
		return errors.New("(stack) " + err.Error())
	}

	//bot.Tgbot.Debug = true

	if bot.Verbose {
		bot.username = bot.Tgbot.Self.UserName
		bot.userID = bot.Tgbot.Self.ID
		log.Printf("(stack) Bot username \"%s\"", bot.Tgbot.Self.UserName)
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

// Version - return bot version
func (bot *Bot) Version() string {
	return version
}

// RegisterProcessor aggiunge un processor al bot, estendendone le funzionalità
func (bot *Bot) RegisterProcessor(name string, processor Processor, configData interface{}) {
	name = strings.Title(name)

	bot.processors = append(bot.processors, processor)
	bot.processorsNames = append(bot.processorsNames, name)

	bot.RegisterConfig(name, configData)
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

	updates, err := bot.Tgbot.GetUpdatesChan(u)
	if err != nil {
		return errors.New("(stack) " + err.Error())
	}

	log.Println("Listening for updates...")

	// bloccante
	for update := range updates {
		// Delega le update ai processori nel modo più trasparente possibile.
		// Il primo che processa interrompe la coda.
		// L'ordine dei processori è inverso; l'ultimo, che è questo oggetto bot,
		// parserà comandi e messaggi (richiamando a sua volta i rispettivi metodi
		// dei processori) soltanto se nessuno ha già processato le update.
		for i := len(bot.processors) - 1; i >= 0; i-- {
			p := bot.processors[i]

			processed, err := p.ProcessUpdate(update)
			if err != nil {
				return err
			}
			if processed {
				break
			}
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
