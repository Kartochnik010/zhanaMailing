package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {

}

var (
	subscribers = []User{}
	publishers  = []User{{Id: "691132073"}, {Id: "1024602209"}, {Id: "632616107"}}
	// mongouri    = os.Getenv("MONGO_URI")
	// botKey      = os.Getenv("TELEGRAM_BOT_KEY")
	mongouri = "mongodb+srv://root:0000@cluster0.qaotl.mongodb.net/novye"
	botKey   = "5626615413:AAEr2z8k4lQoYgEFgjRKo_hn8UNc_N4iiEk"
)

type bot struct {
	api     *tgbotapi.BotAPI
	updates tgbotapi.UpdatesChannel
}

type application struct {
	db *mongo.Database
	bot
}

func main() {
	botAPI, err := tgbotapi.NewBotAPI(botKey)
	if err != nil {
		log.Panic(err)
	}
	// botAPI.Debug = true
	log.Printf("Authorized on account %s", botAPI.Self.UserName)

	updatesConfig := tgbotapi.NewUpdate(0)
	updatesConfig.Timeout = 60
	updates := botAPI.GetUpdatesChan(updatesConfig)

	db, err := openDB(mongouri)
	if err != nil {
		log.Panic(err)
	}
	defer db.Client().Disconnect(context.TODO())

	app := &application{
		db: db,
		bot: bot{
			api:     botAPI,
			updates: updates,
		},
	}
	app.ListenUpdates()
}

func (a *application) ListenUpdates() {
	db := a.db
	bot := a.bot.api
	updates := a.updates
updatesLoop:
	for update := range updates {
		subIdInt := strconv.Itoa(int(update.Message.Chat.ID))
		if update.Message.Text == "/start" {

			err := Insert(db, User{
				Id:        subIdInt,
				UserName:  update.Message.From.UserName,
				FirstName: update.Message.From.FirstName,
				LastName:  update.Message.From.LastName,
			})
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "failed to add subscription"))
				log.Println(err)
				continue updatesLoop
			}
			users, err := GetUsers(db)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "something went wrong"))
				log.Println(err)
				continue updatesLoop
			}

			subscribers = users

			printUsers("subscribers", subscribers)

			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Welcome, "+update.Message.Chat.FirstName+" "+update.Message.Chat.LastName))
			continue updatesLoop
		}

		if update.Message.Text == "/stop" {
			err := Delete(db, subIdInt)

			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "failed to delete subscription"))
				log.Println(err)
				continue updatesLoop
			}
			users, err := GetUsers(db)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "something went wrong"))
				log.Println(err)
				continue updatesLoop
			}
			subscribers = users

			printUsers("subscribers", subscribers)

			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "User "+update.Message.Chat.UserName+" successfully unsubscribed"))
			continue updatesLoop
		}
		if update.Message.Text == "/publish" {
			if !IDisInUsers(subIdInt, publishers) {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "not a publisher"))
				continue updatesLoop
			}
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "what message do you want to publish?"))
			users, err := GetUsers(db)
			if err != nil {
				panic(err)
			} else {
				subscribers = users
				printUsers("subscribers", subscribers)
			}
			newMsg := <-updates

			for _, sub := range subscribers {

				subIdInt, _ := strconv.Atoi(sub.Id)
				_, err = bot.Send(tgbotapi.NewCopyMessage(int64(subIdInt), update.Message.Chat.ID, newMsg.Message.MessageID))
				if err != nil {
					log.Println("failed to send message to", sub)
				}
			}
			continue updatesLoop

			// bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "hi ilyas"))
		}

		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}
	}
}

func printUsers(scope string, users []User) {
	fmt.Println("List of", scope)
	for _, user := range users {
		fmt.Printf("ID: %s, username: %s, Name: %s %s \n", user.Id, user.UserName, user.FirstName, user.LastName)
	}
}

func IDisInUsers(updateMessageChatId string, users []User) bool {
	for _, user := range users {
		if user.Id == updateMessageChatId {
			return true
		}
	}
	return false
}

type User struct {
	Id        string `bson:"_id" json:"_id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	UserName  string `json:"userName"`
}

func openDB(dsn string) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	db, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
	if err != nil {
		return nil, err
	}
	return db.Database("novye"), nil
}

func Insert(DB *mongo.Database, user User) error {
	_, err := DB.Collection("subs").InsertOne(context.TODO(), user)

	return err
}

func Delete(DB *mongo.Database, userId string) error {
	res, err := DB.Collection("subs").DeleteOne(context.TODO(), bson.M{"_id": userId})
	if res.DeletedCount == 0 {
		return errors.New("not found")
	}
	return err
}

func GetUsers(DB *mongo.Database) ([]User, error) {
	cursor, err := DB.Collection("subs").Find(context.Background(), bson.D{})
	if err != nil {
		return nil, err
	}
	var users []User
	if err = cursor.All(context.TODO(), &users); err != nil {
		return nil, err
	}
	return users, nil
}
