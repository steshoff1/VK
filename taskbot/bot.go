package main

// сюда писать код

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	// @BotFather в телеграме даст вам это
	BotToken = "5805187966:AAFhlo_oz4pf5-ojuYc2oLNSJ25hb8RZCpI"

	// урл выдаст вам игрок или хероку
	WebhookURL = "https://0adc-46-138-167-170.ngrok-free.app"
)

type Task struct {
	ID     int
	info   string
	owner  tgbotapi.User
	worker *tgbotapi.User
}

type Tasks struct {
	ID []*Task
}

type Bot struct {
	tasks  Tasks
	taskID int
	funcs  map[string]func(*Bot, string, tgbotapi.User) []tgbotapi.MessageConfig
}

func newBot() Bot {
	newBot := Bot{
		taskID: 0,
		tasks: Tasks{
			ID: make([]*Task, 0),
		},
		funcs: map[string]func(*Bot, string, tgbotapi.User) []tgbotapi.MessageConfig{
			"new":         newTask,
			"tasks":       showTasks,
			"assign_":     assignTask,
			"unassign_":   unassignTask,
			"resolve_":    resolveTask,
			"signedTasks": showAssignedTasks,
			"ableTasks":   showUnAssignedTasks,
			"my":          myTasks,
			"mine":        myTasksCreated,
			"owner":       ownerTasks,
			"start":       makeStartMsg,
		},
	}
	return newBot
}

func (t *Tasks) newTask(info string, ID int, owner tgbotapi.User) {
	newTask := &Task{
		ID:    ID,
		info:  info,
		owner: owner}
	t.ID = append(t.ID, newTask)
}

func newTask(bot *Bot, text string, owner tgbotapi.User) []tgbotapi.MessageConfig {
	if text == "" {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(owner.ID, "Введите команду в виде /new abc, где abc - описание задания")}
	}
	bot.taskID++
	bot.tasks.newTask(text, bot.taskID, owner)
	str := fmt.Sprintf(`Задача "%s" создана, id=%d`, text, bot.taskID)
	msg := tgbotapi.NewMessage(owner.ID, str)
	return []tgbotapi.MessageConfig{msg}
}

func (t *Tasks) findTask(ID int) (*Task, int) {
	for taskID, task := range t.ID {
		if task.ID == ID {
			return task, taskID
		}
	}
	return nil, 0
}

func showTasks(bot *Bot, text string, me tgbotapi.User) []tgbotapi.MessageConfig {
	str := ""
	for i, task := range bot.tasks.ID {
		if i != 0 {
			str += "\n\n"
		}
		str += fmt.Sprintf("%d. %s by @%s", task.ID, task.info, task.owner.UserName)
		switch {
		case task.worker != nil && me.ID == task.worker.ID:
			str += fmt.Sprintf("\nassignee: я\n/unassign_%d /resolve_%d", task.ID, task.ID)

		case task.worker != nil:
			str += fmt.Sprintf("\nassignee: @%s", task.worker)

		default:
			str += fmt.Sprintf("\n/assign_%d", task.ID)

		}
	}

	if str == "" {
		str = "Нет задач"
	}

	msg := tgbotapi.NewMessage(me.ID, str)

	return []tgbotapi.MessageConfig{msg}

}

func showAssignedTasks(bot *Bot, text string, me tgbotapi.User) []tgbotapi.MessageConfig {
	str := ""
	for i, task := range bot.tasks.ID {
		if i != 0 {
			str += "\n\n"
		}
		if task.worker != nil {
			str += fmt.Sprintf("%d. %s by @%s", task.ID, task.info, task.owner.UserName)
			switch {
			case task.worker != nil && me.ID == task.worker.ID:
				str += fmt.Sprintf("\nassignee: я\n/unassign_%d /resolve_%d", task.ID, task.ID)

			case task.worker != nil:
				str += fmt.Sprintf("\nassignee: @%s", task.worker)

			default:
				str += fmt.Sprintf("\n/assign_%d", task.ID)

			}
		}
	}

	if str == "" {
		str = "Нет задач"
	}

	msg := tgbotapi.NewMessage(me.ID, str)

	return []tgbotapi.MessageConfig{msg}

}

func showUnAssignedTasks(bot *Bot, text string, me tgbotapi.User) []tgbotapi.MessageConfig {
	str := ""
	for i, task := range bot.tasks.ID {
		if i != 0 {
			str += "\n\n"
		}
		if task.worker == nil {
			str += fmt.Sprintf("%d. %s by @%s", task.ID, task.info, task.owner.UserName)
			switch {
			case task.worker != nil && me.ID == task.worker.ID:
				str += fmt.Sprintf("\nassignee: я\n/unassign_%d /resolve_%d", task.ID, task.ID)

			case task.worker != nil:
				str += fmt.Sprintf("\nassignee: @%s", task.worker)

			default:
				str += fmt.Sprintf("\n/assign_%d", task.ID)

			}
		}
	}

	if str == "" {
		str = "Нет задач"
	}

	msg := tgbotapi.NewMessage(me.ID, str)

	return []tgbotapi.MessageConfig{msg}

}

// func (t *Tasks) assignTask(taskID int, worker *tgbotapi.User) {
// 	t.ID[taskID].worker = worker
// }

func assignTask(bot *Bot, text string, worker tgbotapi.User) []tgbotapi.MessageConfig {
	ID, err := strconv.Atoi(text)
	if err != nil {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(worker.ID, "Введите команду в виде /assign_1, где 1 - номер задачки")}
	}

	task, _ := bot.tasks.findTask(ID)
	if task == nil {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(worker.ID, "Нет такого задания")}
	}
	if task.worker != nil && task.worker.ID == worker.ID {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(worker.ID, "Задача уже на вас")}
	}

	prevWorker := task.worker
	task.worker = &worker

	str1 := fmt.Sprintf(`Задача "%s" назначена на вас`, task.info)
	if prevWorker == nil {
		if worker.ID == task.owner.ID {
			return []tgbotapi.MessageConfig{
				tgbotapi.NewMessage(worker.ID, str1),
			}
		}
		str2 := fmt.Sprintf(`Задача "%s" назначена на @%s`, task.info, worker.UserName)
		return []tgbotapi.MessageConfig{
			tgbotapi.NewMessage(worker.ID, str1),
			tgbotapi.NewMessage(task.owner.ID, str2),
		}
	}

	str2 := fmt.Sprintf(`Задача "%s" назначена на @%s`, task.info, worker.UserName)
	return []tgbotapi.MessageConfig{
		tgbotapi.NewMessage(worker.ID, str1),
		tgbotapi.NewMessage(prevWorker.ID, str2),
	}

}

func unassignTask(bot *Bot, text string, worker tgbotapi.User) []tgbotapi.MessageConfig {
	ID, err := strconv.Atoi(text)
	if err != nil {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(worker.ID, "Введите команду в виде /unassign_1, где 1 - номер задачки")}
	}

	task, _ := bot.tasks.findTask(ID)
	if task == nil {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(worker.ID, "Нет такого задания")}
	}

	if task.worker.ID != worker.ID {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(worker.ID, "Задача не на вас")}
	}

	defer func(t *Task) {
		t.worker = nil
	}(task)

	if task.worker.ID == task.owner.ID {
		return []tgbotapi.MessageConfig{
			tgbotapi.NewMessage(task.owner.ID, "Принято"),
		}
	}

	str1 := fmt.Sprintf(`Задача "%s" осталась без исполнителя`, task.info)
	str2 := "Принято"
	return []tgbotapi.MessageConfig{
		tgbotapi.NewMessage(task.owner.ID, str1),
		tgbotapi.NewMessage(worker.ID, str2),
	}

}

func resolveTask(bot *Bot, text string, user tgbotapi.User) []tgbotapi.MessageConfig {
	msgID, err := strconv.Atoi(text)
	if err != nil {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(user.ID, "Введите команду в виде /resolve_1, где 1 - номер задания")}
	}

	task, taskID := bot.tasks.findTask(msgID)
	if task == nil {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(user.ID, "Нет такого задания")}
	}

	if task.worker == nil || task.worker.ID != user.ID {
		return []tgbotapi.MessageConfig{tgbotapi.NewMessage(user.ID, "Таск должны выполнять вы")}
	}

	str := fmt.Sprintf(`Задача "%s" выполнена`, task.info)
	bot.tasks.ID = append(bot.tasks.ID[:taskID], bot.tasks.ID[taskID+1:]...)

	str1 := fmt.Sprintf(`Задача "%s" выполнена @%s`, task.info, task.worker)
	return []tgbotapi.MessageConfig{
		tgbotapi.NewMessage(user.ID, str),
		tgbotapi.NewMessage(task.owner.ID, str1),
	}
}

func (t *Tasks) findMine(me tgbotapi.User) []Task {
	res := make([]Task, 0)
	for _, task := range t.ID {
		if task.worker != nil && task.worker.ID == me.ID {
			res = append(res, *task)
		}
	}
	return res
}

func (t *Tasks) findCreatedMine(me tgbotapi.User) []Task {
	res := make([]Task, 0)
	for _, task := range t.ID {
		if task.worker != nil && task.worker.ID == me.ID {
			res = append(res, *task)
		}
	}
	return res
}

func myTasks(bot *Bot, text string, user tgbotapi.User) []tgbotapi.MessageConfig {
	res := ""

	tasks := bot.tasks.findMine(user)
	for i, task := range tasks {
		if i != 0 {
			res += "\n"
		}
		res += fmt.Sprintf("%d. %s by @%s\n/unassign_%d /resolve_%d", task.ID, task.info, task.owner.UserName, task.ID, task.ID)
	}
	if res == "" {
		res = "У вас нет заданий"
	}
	return []tgbotapi.MessageConfig{
		tgbotapi.NewMessage(user.ID, res),
	}

}
func myTasksCreated(bot *Bot, text string, user tgbotapi.User) []tgbotapi.MessageConfig {
	res := ""

	tasks := bot.tasks.findCreatedMine(user)
	for i, task := range tasks {
		if i != 0 {
			res += "\n"
		}
		res += fmt.Sprintf("%d. %s by @%s\n/unassign_%d /resolve_%d", task.ID, task.info, task.owner.UserName, task.ID, task.ID)
	}
	if res == "" {
		res = "У вас нет заданий"
	}
	return []tgbotapi.MessageConfig{
		tgbotapi.NewMessage(user.ID, res),
	}

}

func (t *Tasks) findOwners(user tgbotapi.User) []Task {
	res := make([]Task, 0)
	for _, task := range t.ID {
		if task.owner.ID == user.ID {
			res = append(res, *task)
		}
	}
	return res
}

func ownerTasks(bot *Bot, text string, user tgbotapi.User) []tgbotapi.MessageConfig {
	res := ""
	tasks := bot.tasks.findOwners(user)
	for i, task := range tasks {
		if i != 0 {
			res += "\n"
		}
		res += fmt.Sprintf("%d. %s by @%s\n/assign_%d", task.ID, task.info, task.owner.UserName, task.ID)
	}
	if res == "" {
		res = "У вас нет заданий. Создайте первое!"
	}
	return []tgbotapi.MessageConfig{
		tgbotapi.NewMessage(user.ID, res),
	}

}

func makeStartMsg(bot *Bot, text string, user tgbotapi.User) []tgbotapi.MessageConfig {
	if text != " " {
		text = "Добро пожаловать в бота-планировщика! Чтобы вы хотели сделать?"
	}
	msg := tgbotapi.NewMessage(user.ID, text)
	// создаем первый слой кнопок
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Личные задачи"),
			tgbotapi.NewKeyboardButton("Все задачи"),
			tgbotapi.NewKeyboardButton("Исполнители"),
			tgbotapi.NewKeyboardButton("Работодатели"),
		),
	)
	msg.ReplyMarkup = keyboard
	return []tgbotapi.MessageConfig{msg}

}

func ParseWord(w string) []string {
	ind := strings.Index(w, "_")
	if ind == -1 {
		return []string{w}
	}
	command := w[:ind+1]
	if ind < len(w)-1 {
		return []string{command, w[ind+1:]}
	}
	return []string{command}
}

func ParseString(msg string) ([]string, error) {
	log.Println(msg)
	if msg == "" || msg[0] != '/' {
		return []string{msg}, nil
	}

	ind := strings.Index(msg, " ")
	if ind == -1 {
		strs := ParseWord(msg[1:])
		if len(strs) == 1 {
			strs = append(strs, "")
		}
		return strs, nil
	}

	strs := ParseWord(msg[1:ind])
	if len(strs) == 1 {
		if ind < len(msg)-1 {
			strs = append(strs, msg[ind+1:])
		} else {
			strs = append(strs, "")
		}
	}

	return strs, nil
}

func startTaskBot(ctx context.Context) error {
	// сюда пишите ваш код
	_ = ctx

	nb := newBot()

	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		return err
	}

	bot.Debug = true
	fmt.Printf("Authorized on account %s\n", bot.Self.UserName)

	wh, err := tgbotapi.NewWebhook(WebhookURL)
	if err != nil {
		log.Printf("NewWebhook failed: %s", err)
		return err
	}

	_, err = bot.Request(wh)
	if err != nil {
		log.Fatalf("SetWebhook failed: %s", err)
	}

	http.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all is working"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	go func() {
		log.Fatalln("http err:", http.ListenAndServe(":"+port, nil))
	}()
	log.Println("start listen :" + port)

	updates := bot.ListenForWebhook("/")

	infoFlag := false
	command := ""
	//counter := 0
	for update := range updates {
		flag := false
		keyboard := tgbotapi.NewReplyKeyboard()
		log.Printf("upd: %#v\n", update)

		if update.Message == nil {
			continue
		}

		newMassage, err := ParseString(update.Message.Text)

		log.Println(newMassage)
		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, err.Error())
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			continue
		}

		switch newMassage[0] {
		//первый уровень
		case "Личные задачи":
			//newMassage[0] = "tasks"
			flag = true
			keyboard = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Мои задачи"),
					tgbotapi.NewKeyboardButton("Выполнить задачу"),
					tgbotapi.NewKeyboardButton("Отказаться от задачи"),
					tgbotapi.NewKeyboardButton("Вернуться в главное меню"),
				),
			)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что бы вы хотели сделать?")
			msg.ReplyMarkup = keyboard
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			continue
		case "Мои задачи":
			newMassage[0] = "my"

		case "Выполнить задачу":
			command = "resolve_"
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Какой номер задачки, которую вы хотите выполнить?")
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			infoFlag = true
			continue

		case "Отказаться от задачи":
			command = "unassign_"
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Какой номер задачки, от которой вы хотите отказаться?")
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			infoFlag = true
			continue

		case "Вернуться в главное меню":
			item := makeStartMsg(&nb, "Вы вернулись в главное меню. Что бы хотели сделать?", *update.SentFrom())
			for _, res := range item {
				bot.Send(res)
			}
			continue

		case "Все задачи":
			flag = true
			keyboard = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Посмотреть задачи"),
					tgbotapi.NewKeyboardButton("Создать задачу"),
					tgbotapi.NewKeyboardButton("Свободные задачи"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Взять задачу"),
					tgbotapi.NewKeyboardButton("Отказаться от задачи"),
					tgbotapi.NewKeyboardButton("Вернуться в главное меню")),
			)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что бы вы хотели сделать?")
			msg.ReplyMarkup = keyboard
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			continue

		case "Посмотреть задачи":
			newMassage[0] = "tasks"

		case "Взять задачу":
			command = "assign_"
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Какой номер задачки, которую вы хотите выполнить?")
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			infoFlag = true
			continue
		case "Создать задачу":
			command = "new"
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Опишите задание")
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			infoFlag = true
			continue

		case "Исполнители":
			flag = true
			keyboard = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Мои задачи"),
					tgbotapi.NewKeyboardButton("Исполняемые задачи"),
					tgbotapi.NewKeyboardButton("Вернуться в главное меню"),
				),
			)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что бы вы хотели сделать?")
			msg.ReplyMarkup = keyboard
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			continue

		case "Свободные задачи":
			newMassage[0] = "ableTasks"

		case "Исполняемые задачи":
			newMassage[0] = "signedTasks"

		case "Работодатели":
			flag = true
			keyboard = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Созданные мной задачи"),
					tgbotapi.NewKeyboardButton("Посмотреть задачи пользователя"),
					tgbotapi.NewKeyboardButton("Вернуться в главное меню"),
				),
			)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что бы вы хотели сделать?")
			msg.ReplyMarkup = keyboard
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			continue
		case "Созданные мной задачи":
			newMassage[0] = "mine"

		case "Посмотреть задачи пользователя":
			command = "owner"
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите имя интересующего вас пользователя:")
			_, err := bot.Send(msg)
			if err != nil {
				log.Println(err)
			}
			infoFlag = true
			continue
		}

		if infoFlag {
			newMassage = []string{command, update.Message.Text}
			infoFlag = false
		}

		if len(newMassage) < 2 {
			newMassage = append(newMassage, "")
		}

		handler, ok := nb.funcs[newMassage[0]]
		if !ok {
			msg := tgbotapi.NewMessage(
				update.Message.Chat.ID,
				`Нет такой команды`,
			)
			bot.Send(msg)
			continue
		}

		item := handler(&nb, newMassage[1], *update.SentFrom())
		for _, res := range item {
			if flag {
				res.ReplyMarkup = keyboard
			}
			bot.Send(res)

		}

	}
	return nil

}

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		panic(err)
	}
}
