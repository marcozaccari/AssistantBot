package bot

import (
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// MessageHandler contiene i riferimenti utili all'invio messaggi dai metodi di processing
type MessageHandler struct {
	userID   int
	username string
	group    userGroup

	chatID    int64
	isPrivate bool

	messageID     int // messaggio a cui appartiene il comando
	editMessageID int // se > 0 messaggio da editare anzichè inviarne uno nuovo

	replyUserID   int // utente del messaggio a cui si è risposto
	replyUsername string
}

// MessageResponseOpt contiene i flag di modalità di risposta
type MessageResponseOpt struct {
	ForcePrivate         bool
	ReplyToSenderMessage bool
	ReplaceSenderMessage bool
	LinksPreview         bool
	HTMLformat           bool
}

// ogni quanti messaggi inviati (x2) deve pulire la prima metà di lookup*
const gcMaxSentMessages = 100

type sentMessagesLookups struct {
	locker sync.Mutex

	// chiave: ID messaggio mittente; valore: ID messaggio inviato
	lookupSenderSent map[int]int
	// valore: ID messaggio mittente
	lookupSent  []int
	sentCounter int
}

// NewMessageResponseOpt resituisce un MessageResponse con valori inizializzati
func (bot *Bot) NewMessageResponseOpt() MessageResponseOpt {
	return MessageResponseOpt{
		ReplyToSenderMessage: true,
		LinksPreview:         true,
		HTMLformat:           true,
	}
}

// SendMessageResponse invia un messaggio di risposta all'handler
func (bot *Bot) SendMessageResponse(handler MessageHandler, text string, opt MessageResponseOpt) {
	var chatID int64

	if opt.ForcePrivate && !handler.isPrivate {
		u, ok := bot.getUserByID(handler.userID)
		if !ok {
			log.Println("Cannot send to private chat", handler, text)
			return
		}
		chatID = u.PrivateChatID
	} else {
		chatID = handler.chatID
	}

	var replyMessageID int
	if opt.ReplyToSenderMessage && (chatID == handler.chatID) && !opt.ReplaceSenderMessage {
		replyMessageID = handler.messageID
	}

	// Send

	if handler.editMessageID > 0 {
		msg := tgbotapi.NewEditMessageText(chatID, handler.editMessageID, text)
		if opt.HTMLformat {
			msg.ParseMode = "HTML"
		} else {
			msg.ParseMode = "MarkdownV2"
		}
		msg.DisableWebPagePreview = !opt.LinksPreview

		bot.tgbot.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyToMessageID = replyMessageID
		if opt.HTMLformat {
			msg.ParseMode = "HTML"
		} else {
			msg.ParseMode = "MarkdownV2"
		}

		msg.DisableWebPagePreview = !opt.LinksPreview

		newmsg, _ := bot.tgbot.Send(msg)

		if opt.ReplaceSenderMessage {
			cfg := tgbotapi.DeleteMessageConfig{
				ChatID:    handler.chatID,
				MessageID: handler.messageID,
			}

			bot.tgbot.DeleteMessage(cfg)
		} else {
			// indicizza il messaggio inviato nella lookup
			bot.sentMessages.locker.Lock()
			defer bot.sentMessages.locker.Unlock()

			bot.sentMessages.lookupSenderSent[handler.messageID] = newmsg.MessageID
			bot.sentMessages.lookupSent[bot.sentMessages.sentCounter] = handler.messageID

			bot.sentMessages.sentCounter++
			// copie esplicite necessarie perchè le map non ritornano memoria dopo i delete
			if bot.sentMessages.sentCounter == gcMaxSentMessages*2 {
				//fmt.Println("lookup", bot.sentMessages.sentCounter, bot.sentMessages.lookupSent, bot.sentMessages.lookupSenderSent)

				// copia la seconda metà di gcMaxSentMessages
				newmap := make(map[int]int)
				for _, v := range bot.sentMessages.lookupSent[gcMaxSentMessages:] {
					newmap[v] = bot.sentMessages.lookupSenderSent[v]
				}
				bot.sentMessages.lookupSenderSent = newmap

				newarr := make([]int, gcMaxSentMessages*2)
				copy(newarr, bot.sentMessages.lookupSent[gcMaxSentMessages:])
				bot.sentMessages.lookupSent = newarr

				bot.sentMessages.sentCounter = gcMaxSentMessages
				//fmt.Println("lookup post", bot.sentMessages.sentCounter, bot.sentMessages.lookupSent, bot.sentMessages.lookupSenderSent)
			}
		}
	}
}

// SendMessageResponseToPrivate invia un messaggio di risposta all'handler forzandolo in chat privata
func (bot *Bot) SendMessageResponseToPrivate(handler MessageHandler, text string, opt MessageResponseOpt) {
	opt.ForcePrivate = true
	bot.SendMessageResponse(handler, text, opt)

	if !handler.isPrivate {
		opt.ForcePrivate = false
		opt.ReplyToSenderMessage = true
		bot.SendMessageResponse(handler, "pvt", opt)
	}
}

func (bot *Bot) initMessages() {
	bot.sentMessages.lookupSenderSent = make(map[int]int)
	bot.sentMessages.lookupSent = make([]int, gcMaxSentMessages*2)
}

func (bot *Bot) ProcessMessage(handler MessageHandler, text string) (bool, error) {
	return false, nil
}
