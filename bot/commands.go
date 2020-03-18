package bot

// Resetta l'owner del bot; pretende secureToken come parametro
const superCommandOwner = "owner"

// gestisce i permessi
const commandUser = "user"

type CommandHandler struct {
	userID int

	chatID    int64
	isPrivate bool

	messageID int // messaggio a cui appartiene il comando

	replyUserID   int // utente del messaggio a cui si è risposto
	replyUsername string
}
