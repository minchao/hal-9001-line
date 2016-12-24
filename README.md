# hal-9001-line

A [Hal-9001][1] broker for LINE

## Installation

```
go get -u github.com/minchao/hal-9001-line
```

## Using line broker

```
import "github.com/minchao/hal-9001-line/line"
```

Configure the line broker in the main.go of chatbot

```
func main() {
    // ...
    conf := line.Config{
        Secret: os.Getenv("LINE_SECRET"),
        Token:  os.Getenv("LINE_TOKEN"),
        Listen: ":"+os.Getenv("LINE_WEBHOOK_PORT"),
    }
    
    router := hal.Router()
    router.AddBroker(conf.NewBroker("line"))
    // ...
}
```

## Webhook URL

LINE broker use the [webhook][2] URL to receive notifications in real-time. For actual use, you must support HTTPS, using a reverse proxy or something else.

[1]: https://github.com/Netflix/hal-9001
[2]: https://devdocs.line.me/en/#webhooks