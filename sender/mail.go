package sender

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"os"
	"watchdog"

	"gopkg.in/gomail.v2"
)

type MailSender struct {
	Account    string   `json:"account"`
	Password   string   `json:"password"`
	SMTPServer string   `json:"smtp_server"`
	Port       int      `json:"port"`
	PubList    []string `json:"pub_list"`
}

const TemplateFileName = "template.html"

var msgTemplate *template.Template

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

func NewMailSender() Sender {
	return &MailSender{}
}

func (s *MailSender) Init(conf io.Reader) error {
	dc := json.NewDecoder(conf)
	return dc.Decode(s)
}

func (s *MailSender) Listen(msq chan interface{}) {
	for m := range msq {
		s.send(m.(*watchdog.Record))
	}
}

func (s *MailSender) send(body *watchdog.Record) error {
	m := gomail.NewMessage()
	buf := new(bytes.Buffer)
	msgTemplate.Execute(buf, body)
	m.SetHeader("From", s.Account)
	m.SetHeader("To", s.PubList...)
	m.SetHeader("Subject", "WatchDog Alert")
	m.SetBody("text/html", buf.String())
	d := gomail.NewDialer(s.SMTPServer, s.Port, s.Account, s.Password)
	return d.DialAndSend(m)
}
