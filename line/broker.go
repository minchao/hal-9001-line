package line

import (
	"log"
	"net/http"
	"time"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/netflix/hal-9001/hal"
)

type Broker struct {
	Client *linebot.Client
	Config Config
	inst   string
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
	}
}

func (b Broker) Name() string {
	return b.inst
}

func (b Broker) Send(evt hal.Evt) {
	event := evt.Original.(*linebot.Event)
	if _, err := b.Client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(evt.Body)).Do(); err != nil {
		log.Printf("Failed to send message: %s\n", err)
	}
}

func (b Broker) SendTable(evt hal.Evt, header []string, rows [][]string) {

}

func (b Broker) SendDM(evt hal.Evt) {

}

func (b Broker) SetTopic(roomId, topic string) error {
	return nil
}

func (b Broker) GetTopic(roomId string) (topic string, err error) {
	return "", nil
}

func (b Broker) LooksLikeRoomId(room string) bool {
	return false
}

func (b Broker) LooksLikeUserId(user string) bool {
	return false
}

func (b Broker) RoomIdToName(id string) (name string) {
	return ""
}

func (b Broker) RoomNameToId(name string) (id string) {
	return ""
}

func (b Broker) UserIdToName(id string) (name string) {
	return ""
}

func (b Broker) UserNameToId(name string) (id string) {
	return ""
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
			log.Fatal(err)
		}
	}()

	for event := range incoming {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				out <- &hal.Evt{
					ID:       message.ID,
					Body:     message.Text,
					Room:     event.Source.RoomID,
					RoomId:   event.Source.RoomID,
					User:     event.Source.UserID,
					UserId:   event.Source.UserID,
					Time:     time.Now(),
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
