package watchdog

import (
	"bytes"
	"encoding/json"
	"html/template"
	"os"

	"gopkg.in/gomail.v2"
)

const MessageTemplate = `<h1>WATCH OUT!</h1>
Client <b>{{.Name}}</b> has lost connection for <u>{{.Dura}}</u> since<br/>
<i>{{.LastComm}}</i>`

var msg = template.Must(template.New("msg").Parse(MessageTemplate))

type MailConf struct {
	Account    string
	Password   string
	SMTPServer string
	Port       int
	PubList    []string
}

func ReadMailConf(file string) (*MailConf, error) {
	confFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	dc := json.NewDecoder(confFile)
	conf := MailConf{}
	dc.Decode(&conf)
	return &conf, nil
}

func sendMail(conf *MailConf, body *AlertBody) error {
	m := gomail.NewMessage()
	buf := new(bytes.Buffer)
	msg.Execute(buf, body)
	m.SetHeader("From", conf.Account)
	m.SetHeader("To", conf.PubList...)
	m.SetHeader("Subject", "WatchDog Alert")
	m.SetBody("text/html", buf.String())
	d := gomail.NewDialer(conf.SMTPServer, conf.Port, conf.Account, conf.Password)
	return d.DialAndSend(m)
}
