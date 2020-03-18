package bot

import "time"

type configData struct {
	SecureToken       string
	RecoverOldUpdates bool // al riavvio processa le update ancora appese dall'ultimo shutdown

	CommandWord string

	OwnerID int
	Users   []user
}

const saveAfter = 5 * time.Second

func (bot *Bot) SaveConfig() {
	bot.configCtrl.SaveSettingsDebounce(saveAfter)
}
