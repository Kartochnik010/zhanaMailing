package main

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	// subs        = []User{{632616107}, {1024602209}}
	subs        = []User{}
	allowedUser = 632616107
	dns         = "mongodb+srv://root:0000@cluster0.qaotl.mongodb.net/novye"
	botKey      = "5626615413:AAEr2z8k4lQoYgEFgjRKo_hn8UNc_N4iiEk"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(botKey)
	if err != nil {
		log.Panic(err)
	}

	db := mustOpenDB(dns)
	defer db.Client().Disconnect(context.TODO())

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// A loop that is waiting for a message from the user.
	for update := range updates {
		if update.Message.Text == "/start" {
			Insert(db, int(update.Message.Chat.ID))
			users, err := GetUsers(db)
			if err != nil {
				panic(err)
			} else {
				subs = users
				fmt.Println(subs)
			}
		}
		// it will print the username and the message.
		if int(update.Message.Chat.ID) == allowedUser {
			if update.Message.Text == "/publish" {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "what message do you want to publish?"))
				newMsg := <-updates

				for _, sub := range subs {
					_, err := bot.Send(tgbotapi.NewCopyMessage(int64(sub.Id), update.Message.Chat.ID, newMsg.Message.MessageID))
					if err != nil {
						log.Println("failed to send message to", sub)
					}
				}
				continue
			}
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

type User struct {
	Id int `bson:"_id" json:"_id"`
}

func mustOpenDB(dsn string) *mongo.Database {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	db, err := mongo.Connect(ctx, options.Client().ApplyURI(dsn))
	if err != nil {
		panic(err)
	}
	return db.Database("novye")
}

func Insert(DB *mongo.Database, userId int) error {
	user := &User{Id: userId}
	_, err := DB.Collection("subs").InsertOne(context.TODO(), user)

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
