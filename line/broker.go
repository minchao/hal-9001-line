package line

import (
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/netflix/hal-9001/hal"
)

var log hal.Logger

type Broker struct {
	Client *linebot.Client
	Config Config
	inst   string
	i2u    map[string]string // id->name cache
	mutex  sync.Mutex
}

type Config struct {
	Secret string
	Token  string
	Listen string
}

func (c Config) NewBroker(name string) Broker {
	client, err := linebot.New(
		c.Secret,
		c.Token,
	)
	if err != nil {
		log.Fatalf("Could not create the linebot client: %s\n", err)
	}
	return Broker{
		Client: client,
		Config: c,
		inst:   name,
		i2u:    make(map[string]string),
	}
}

func (b Broker) reply(evt hal.Evt) error {
	switch event := evt.Original.(type) {
	case *linebot.Event:
		if _, err := b.Client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(evt.Body)).Do(); err != nil {
			return err
		}
	default:
		return errors.New("Missing original event")
	}
	return nil
}

func (b Broker) Name() string {
	return b.inst
}

func (b Broker) Send(evt hal.Evt) {
	if err := b.reply(evt); err != nil {
		log.Printf("Failed to send message: %s\n", err)
	}
}

func (b Broker) SendTable(evt hal.Evt, header []string, rows [][]string) {
	body := ""
	body += strings.Join(header, " │ ") + "\n"
	body += "──────────────────\n"
	for _, row := range rows {
		body += strings.Join(row, " │ ") + "\n"
	}
	out := evt.Clone()
	out.Body = body
	if err := b.reply(out); err != nil {
		log.Printf("Failed to send table message: %s\n", err)
	}
}

func (b Broker) SendDM(evt hal.Evt) {
	if _, err := b.Client.PushMessage(evt.UserId, linebot.NewTextMessage(evt.Body)).Do(); err != nil {
		log.Printf("Failed to send direct message: %s\n", err)
	}
}

func (b Broker) SetTopic(roomId, topic string) error {
	log.Println("line/SetTopic() is a stub")
	return nil
}

func (b Broker) GetTopic(roomId string) (topic string, err error) {
	log.Println("line/GetTopic() is a stub")
	return "", nil
}

func (b Broker) Leave(roomId string) error {
	_, err := b.Client.LeaveRoom(roomId).Do()
	return err
}

func (b Broker) LooksLikeRoomId(room string) bool {
	log.Println("line/LooksLikeRoomId() is a stub that always return true!")
	return true
}

func (b Broker) LooksLikeUserId(user string) bool {
	log.Println("line/LooksLikeUserId() is a stub that always return true!")
	return true
}

func (b Broker) RoomIdToName(id string) (name string) {
	log.Println("line/RoomIdToName() is a stub that always return with input id!")
	return id
}

func (b Broker) RoomNameToId(name string) (id string) {
	log.Println("line/RoomNameToId() is a stub that always return with input name!")
	return name
}

func (b Broker) UserIdToName(id string) (name string) {
	b.mutex.Lock()
	name, exists := b.i2u[id]
	b.mutex.Unlock()

	if exists {
		return name
	} else {
		profile, err := b.Client.GetProfile(id).Do()
		if err != nil {
			log.Printf("line could not retrieve user profile for '%s': %s\n", id, err)
			return ""
		}

		b.mutex.Lock()
		defer b.mutex.Unlock()

		b.i2u[id] = profile.DisplayName

		return profile.DisplayName
	}
}

func (b Broker) UserNameToId(name string) (id string) {
	log.Println("line/UserNameToId() is a stub that always return with input name!")
	return name
}

func (b Broker) Stream(out chan *hal.Evt) {
	incoming := make(chan *linebot.Event)

	go func() {
		// Setup HTTP Server for receiving requests from LINE platform
		http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
			events, err := b.Client.ParseRequest(req)
			if err != nil {
				if err == linebot.ErrInvalidSignature {
					w.WriteHeader(400)
				} else {
					w.WriteHeader(500)
				}
				return
			}
			for _, event := range events {
				incoming <- event
			}
		})
		// For actual use, you must support HTTPS by using `ListenAndServeTLS`, a reverse proxy or something else.
		if err := http.ListenAndServe(b.Config.Listen, nil); err != nil {
			log.Fatalf("http.ListenAndServe error: %v", err)
		}
	}()

	for event := range incoming {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				var roomId string
				switch event.Source.Type {
				case linebot.EventSourceTypeGroup:
					roomId = event.Source.GroupID
				case linebot.EventSourceTypeRoom:
					roomId = event.Source.RoomID
				case linebot.EventSourceTypeUser:
					roomId = event.Source.UserID
				}

				out <- &hal.Evt{
					ID:       message.ID,
					Body:     message.Text,
					Room:     roomId,
					RoomId:   roomId,
					User:     b.UserIdToName(event.Source.UserID),
					UserId:   event.Source.UserID,
					Time:     event.Timestamp,
					Broker:   b,
					IsChat:   true,
					Original: event,
				}
			default:
				log.Printf("Unhandled message of type '%T': %s ", message, message)
			}
		}
	}
}
