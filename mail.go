package watchdog

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"os"
	"time"

	"gopkg.in/gomail.v2"
)

const TemplateFileName = "template.html"

var msgTemplate *template.Template

type MailBody struct {
	Name     string
	Dura     time.Duration
	LastComm time.Time
}

func init() {
	f, err := os.Open(TemplateFileName)
	if err != nil {
		log.Fatal(err)
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(f)
	if err != nil {
		log.Fatal(err)
	}
	msgTemplate = template.Must(template.New("msg").Parse(buf.String()))
}

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

func sendMail(conf *MailConf, body *MailBody) error {
	m := gomail.NewMessage()
	buf := new(bytes.Buffer)
	msgTemplate.Execute(buf, body)
	m.SetHeader("From", conf.Account)
	m.SetHeader("To", conf.PubList...)
	m.SetHeader("Subject", "WatchDog Alert")
	m.SetBody("text/html", buf.String())
	d := gomail.NewDialer(conf.SMTPServer, conf.Port, conf.Account, conf.Password)
	return d.DialAndSend(m)
}
