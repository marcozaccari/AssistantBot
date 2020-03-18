package bot

import (
	"fmt"
	"log"
	"strconv"
)

type user struct {
	ID            int
	Username      string
	PrivateChatID int64
}

type usersLookupMap map[int]*user

func (bot *Bot) getUserByID(ID int) (u *user, ok bool) {
	u, ok = bot.lookupUsers[ID]
	return
}

func (bot *Bot) getUserByUsername(username string) (u *user, ok bool) {
	for _, u := range bot.lookupUsers {
		if u.Username == username {
			return u, true
		}
	}

	return nil, false
}

func (bot *Bot) addUser(u user, failIfExists bool) bool {
	bot.configLock.Lock()
	defer bot.configLock.Unlock()

	_, ok := bot.lookupUsers[u.ID]
	if ok {
		if failIfExists {
			return false
		}
		*bot.lookupUsers[u.ID] = u
	} else {
		bot.config.Users = append(bot.config.Users, u)
		bot.lookupUsers[u.ID] = &bot.config.Users[len(bot.config.Users)-1]
	}

	bot.SaveConfig()
	return true
}

func (bot *Bot) deleteUser(userID int) bool {
	_, ok := bot.lookupUsers[userID]
	if !ok {
		return false
	}

	bot.configLock.Lock()
	defer bot.configLock.Unlock()

	for i, u := range bot.config.Users {
		if u.ID == userID {
			bot.config.Users[i] = bot.config.Users[len(bot.config.Users)-1]
			bot.config.Users = bot.config.Users[:len(bot.config.Users)-1]
			bot.initUsers(false)
			break
		}
	}

	bot.SaveConfig()
	return true
}

func (bot *Bot) resetOwner(newOwnerID int, OwnerUsername string, privateChatID int64) {
	bot.configLock.Lock()

	log.Println("Reset owner to", newOwnerID)

	bot.config.OwnerID = newOwnerID
	bot.configLock.Unlock()

	bot.SaveConfig()

	_, ok := bot.getUserByID(newOwnerID)
	if !ok {
		u := user{
			ID:            newOwnerID,
			Username:      OwnerUsername,
			PrivateChatID: privateChatID,
		}

		bot.addUser(u, false)
	}
}

func (bot *Bot) updateUserData(u *user, newu user) {
	bot.configLock.Lock()
	defer bot.configLock.Unlock()

	*u = newu

	if bot.Debug {
		log.Println("Update user data", *u)
	}

	bot.SaveConfig()
}

func (bot *Bot) updateUsername(u *user, username string) {
	nu := *u
	nu.Username = username

	bot.updateUserData(u, nu)
}

func (bot *Bot) updateUserPrivateChatID(u *user, privateChatID int64) {
	nu := *u
	nu.PrivateChatID = privateChatID

	bot.updateUserData(u, nu)
}

func (bot *Bot) initUsers(verbose bool) {
	// make lookup
	bot.lookupUsers = make(map[int]*user)

	verboseIDs := make([]int, 0)

	for i, user := range bot.config.Users {
		bot.lookupUsers[user.ID] = &bot.config.Users[i]
		verboseIDs = append(verboseIDs, user.ID)
	}

	if verbose {
		log.Println("Users IDs", verboseIDs)
	}
}

func (bot *Bot) processUserCommand(handler CommandHandler, params []string) error {
	showHelp := func() {
		help :=
			"<code>user</code> command usage:\n" +
				"  <code>list</code>  Show users list\n" +
				"  <code>add [id]</code>  Add user to whitelist\n" +
				"  <code>remove [id|username]</code>  Remove user from whitelist\n" +
				"\nHint: <code>id</code> could be avoided by replying to a user's message"

		bot.SendCommandResponseToPrivate(handler, help)
	}

	getUser := func() (int, string, string) {
		var userID int
		var username string
		var errorStr string

		if len(params) > 1 {
			var err error
			userID, err = strconv.Atoi(params[1])
			if err != nil {
				name := params[1]
				if name[0] == '@' {
					name = name[1:]
				}

				u, ok := bot.getUserByUsername(name)
				if ok {
					userID = u.ID
				} else {
					errorStr = "Invalid UserID"
				}
			}

		} else {

			switch handler.replyUserID {
			case 0:
				showHelp()

			default:
				userID = int(handler.replyUserID)
				username = handler.replyUsername
			}

		}

		if userID == bot.tgbot.Self.ID {
			userID = 0
			errorStr = "lol"
		}

		return userID, username, errorStr
	}

	if len(params) < 1 {
		showHelp()
		return nil
	}

	var response string
	var userID int
	var username string

	switch params[0] {
	case "add":
		userID, username, response = getUser()

		if userID > 0 {
			if userID == handler.userID {
				response = "Your are already in whitelist"
			} else {
				u := user{
					ID:       userID,
					Username: username,
				}

				if bot.addUser(u, true) {
					response = fmt.Sprintf("User <b>%v %v</b> added to whitelist", userID, username)
				} else {
					response = fmt.Sprintf("User <b>%v</b> already in whitelist", userID)
				}
			}
		}

	case "remove":
		userID, _, response = getUser()

		if userID > 0 {
			if userID == bot.config.OwnerID {
				response = "Cannot remove my owner"
			} else {
				if bot.deleteUser(userID) {
					response = fmt.Sprintf("User <b>%v</b> deleted from whitelist", userID)
				} else {
					response = fmt.Sprintf("User <b>%v</b> not in whitelist", userID)
				}
			}
		}

	case "list":
		response = "Users list:\n\n"

		for _, u := range bot.lookupUsers {
			response += fmt.Sprint(u.ID)

			if u.Username != "" {
				response += "\t<b>" + u.Username + "</b>"
			}

			if u.PrivateChatID == 0 {
				response += " <i>(pending)</i>"
			} else {
				response += " (active)"
			}

			response += "\n"
			//log.Println(response)
		}

	default:
		showHelp()
	}

	if response != "" {
		bot.SendCommandResponse(handler, response, false)
	}

	return nil
}
