package bot

import (
	"fmt"
	"log"
	"strconv"
)

type userGroup string

const (
	groupOwner userGroup = "owner"
	groupAdmin userGroup = "admin"
)

type user struct {
	ID            int
	Username      string
	Email         string
	Group         userGroup
	PrivateChatID int64
}

type usersLookupMap map[int]*user

func (bot *Bot) GetUserEmail(userID int) (email string, ok bool) {
	bot.configLock.Lock()
	defer bot.configLock.Unlock()

	u, ok := bot.lookupUsers[userID]
	if !ok {
		return "", false
	}

	return u.Email, true
}

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
			bot.initUsers()
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

	u := user{
		ID:            newOwnerID,
		Username:      OwnerUsername,
		Group:         groupOwner,
		PrivateChatID: privateChatID,
	}

	bot.addUser(u, false)
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

func (bot *Bot) initUsers() {
	// make lookup
	bot.lookupUsers = make(map[int]*user)

	verboseIDs := make([]int, 0)

	for i, user := range bot.config.Users {
		bot.lookupUsers[user.ID] = &bot.config.Users[i]
		verboseIDs = append(verboseIDs, user.ID)
	}

	if bot.Verbose {
		log.Println("Users IDs", verboseIDs)
	}
}

func (bot *Bot) processUserCommand(handler MessageHandler, params []string) error {
	if handler.Group != groupOwner && handler.Group != groupAdmin {
		if bot.Verbose {
			log.Println("No permission for command")
		}
		return nil
	}

	showHelp := func() {
		help :=
			"<code>user</code> command parameters:\n" +
				"  <code>list</code>  Show users list\n" +
				"  <code>add {id}</code>  Add user to whitelist\n" +
				"  <code>remove {id|username}</code>  Remove user from whitelist\n" +
				"  <code>group [none|admin] {id|username}</code>  Change user's group\n" +
				"  <code>email {address} {id|username}</code>  Change user's e-mail (\"none\" = unset)\n" +
				"\nHint: <code>{id}</code> could be avoided by replying to a user's message"

		opt := bot.NewMessageResponseOpt()
		bot.SendMessageResponseToPrivate(handler, help, opt)
	}

	parseUser := func(paramIdx int) (int, string, string) {
		var userID int
		var username string
		var errorStr string

		if len(params) > paramIdx {
			var err error
			userID, err = strconv.Atoi(params[paramIdx])
			if err != nil {
				name := params[paramIdx]
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

			switch handler.ReplyUserID {
			case 0:
				showHelp()

			default:
				userID = int(handler.ReplyUserID)
				username = handler.ReplyUsername
			}

		}

		if userID == bot.Tgbot.Self.ID {
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
		userID, username, response = parseUser(1)

		if userID > 0 {
			if userID == handler.UserID {
				response = "Your are already in whitelist"
			} else {
				u := user{
					ID:       userID,
					Username: username,
				}

				if bot.addUser(u, true) {
					response = fmt.Sprintf("User <code>%v %v</code> added to whitelist", userID, username)
				} else {
					response = fmt.Sprintf("User <code>%v</code> already in whitelist", userID)
				}
			}
		}

	case "remove":
		userID, _, response = parseUser(1)

		if userID > 0 {
			if userID == bot.config.OwnerID {
				response = "Cannot remove my owner"
			} else {
				if bot.deleteUser(userID) {
					response = fmt.Sprintf("User <code>%v</code> deleted from whitelist", userID)
				} else {
					response = fmt.Sprintf("User <code>%v</code> not in whitelist", userID)
				}
			}
		}

	case "group":
		if len(params) < 2 {
			showHelp()
			return nil
		}
		group := params[1]
		if group != "none" && group != "admin" {
			showHelp()
			return nil
		}

		userID, username, response = parseUser(2)

		if userID > 0 {
			u, ok := bot.getUserByID(userID)
			if !ok {
				response = fmt.Sprintf("User <code>%v</code> not exists", userID)
				break
			}

			if group == "none" {
				u.Group = ""
			} else {
				u.Group = userGroup(group)
			}
			bot.addUser(*u, false)

			response = fmt.Sprintf("User <code>%v %v</code> set to <code>%v</code>", userID, username, group)
		}

	case "email":
		if len(params) < 2 {
			showHelp()
			return nil
		}
		email := params[1]

		userID, username, response = parseUser(2)

		if userID > 0 {
			u, ok := bot.getUserByID(userID)
			if !ok {
				response = fmt.Sprintf("User <code>%v</code> not exists", userID)
				break
			}

			if email == "none" {
				u.Email = ""
				email = "(none)"
			} else {
				u.Email = email
			}

			bot.addUser(*u, false)

			response = fmt.Sprintf("User <code>%v %v</code> email: <code>%v</code>", userID, username, email)
		}

	case "list":
		response = "Users list:\n\n"

		for _, u := range bot.lookupUsers {
			response += "<code>" + fmt.Sprint(u.ID) + "</code>"

			if u.Username != "" {
				response += " <b>" + u.Username + "</b>"
			}

			if u.Email != "" {
				response += " " + u.Email
			}

			switch u.Group {
			case groupOwner:
				response += " <code>★</code>"
			case groupAdmin:
				response += " <code>☆</code>"
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
		opt := bot.NewMessageResponseOpt()
		bot.SendMessageResponseToPrivate(handler, response, opt)
	}

	return nil
}
