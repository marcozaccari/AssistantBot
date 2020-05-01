package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

// StubProcessor implementa tutti i metodi di Processor ma con funzioni vuote,
// pronte per l'override
type StubProcessor struct {
}

func (p *StubProcessor) Help() string {
	/* return "/hello  Print hello world\n" */
	return ""
}

func (p *StubProcessor) Version() string {
	return "0.0.1"
}

func (p *StubProcessor) ProcessUpdate(update tgbotapi.Update) (bool, error) {
	return false, nil
}

func (p *StubProcessor) ProcessCommand(handler MessageHandler, command string, params []string) (bool, error) {
	/*	if command == "hello" {
			message := "Hello World!"

			opt := myBot.NewMessageResponseOpt()
			myBot.SendMessageResponse(handler, message, opt)

			return true, nil
		}
	*/

	return false, nil
}

func (p *StubProcessor) ProcessMessage(handler MessageHandler, text string) (bool, error) {
	return false, nil
}
