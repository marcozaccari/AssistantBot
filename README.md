# AssistantBot
Telegram Bot library usable as personal or team assistant

Usage example:
```go

import "modulo.srl/AssistantBot/bot"

type myBot struct {
	bot.Bot
}

func (bot *myBot) ProcessCommand(handler bot.CommandHandler, command string, params []string) error {

	switch command {
	case "hello":
		bot.SendCommandResponse(handler, "Hello!", false)
	}

	return nil
}

func main() {
	telegramBot := myBot{}
	err = telegramBot.Init(configFilename, true, false)
	if err != nil {
		log.Println("ERROR:", err.Error())
		return
	}

	telegramBot.Do(&telegramBot)
}
```