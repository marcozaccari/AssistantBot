package main

import (
	"log"
	"fmt"

	"modulo.srl/AssistantBot/bot"
)

var tbot *bot.Bot

type myConfig struct {
	Foo int
	Bar string
}

type myProcessor struct {
	config myConfig
}

func (p *myProcessor) ProcessCommand(handler bot.MessageHandler, command string, params []string) (bool, error) {
	if command == "hello" {
		message := fmt.Sprintln("Hello World!", p.config)

		opt := tbot.NewMessageResponseOpt()
		tbot.SendMessageResponse(handler, message, opt)

		return true, nil
	}

	return false, nil
}

func (p *myProcessor) ProcessMessage(handler bot.MessageHandler, text string) (bool, error) {
	message := "ECHO: " + text

	opt := tbot.NewMessageResponseOpt()
	tbot.SendMessageResponse(handler, message, opt)

	return true, nil
}

func (p *myProcessor) Help() string {
	return "/hello  Print hello world\n"
}

func main() {
	processor := myProcessor{}

	tbot = bot.NewBot("settings.bot.json", true, false)

	tbot.RegisterProcessor("myscope", &processor, &processor.config)

	err := tbot.Do()
	if err != nil {
		log.Println("ERROR:", err.Error())
		return
	}
}
