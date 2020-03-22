package bot

import (
	"time"
)

type configData struct {
	SecureToken       string
	RecoverOldUpdates bool // al riavvio processa le update ancora appese dall'ultimo shutdown

	CommandWord string // se di un solo carattere lavora come "/comando"

	ProcessGroupMessages bool

	SilenceTimeoutMins int

	OwnerID int
	Users   []user
}

const saveAfter = 5 * time.Second

// SaveConfig - può essere invocata spesso dato che utilizza l'antibounce
func (bot *Bot) SaveConfig() {
	bot.configCtrl.SaveSettingsDebounce(saveAfter)
}

// SaveConfigNow - salva immediatamente le impostazioni
func (bot *Bot) SaveConfigNow() error {
	return bot.configCtrl.SaveSettings()
}

// LoadConfig - carica le impostazioni.
// Se non viene invocata esternamente ci pensa comunque bot.Do()
func (bot *Bot) LoadConfig() error {
	err := bot.configCtrl.LoadSettings()
	if err != nil {
		return err
	}

	bot.configLoaded = true

	return nil
}

func (bot *Bot) RegisterConfig(scope string, configData interface{}) {
	bot.configs[scope] = configData
}
