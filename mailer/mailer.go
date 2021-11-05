/*
Copyright © 2021 Anton Kaiukov <batko@batko.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package mailer

import (
	"errors"
	"fmt"
	"log"
	"net/smtp"

	"github.com/batk0/gc-tracker/config"
)

const subjectText = "Notification from GC-Tracker"
const greetingText = "Hello!"
const footerText = "This is automated message. Please do not reply."

// Need for mocking
type smtpMailer struct{}
type mail interface {
	SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

func (m *smtpMailer) SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, a, from, to, msg)
}

// wraps sendReal to pass mailer interface
func Send(to string, msg string) error {
	var mailer smtpMailer = smtpMailer{}
	return sendReal(&mailer, to, msg)
}

func getHeaders(from, to, subject string) string {
	return fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n", from, to, subject)
}
func getBody(msg string) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s\n", greetingText, msg, footerText)
}

func sendReal(mailer mail, to string, msg string) error {
	if to == "" {
		return errors.New("emtpty recepients list")
	}
	if msg == "" {
		return errors.New("empty message")
	}
	toList := []string{to}
	header := getHeaders(config.Config.SMTPUser, to, subjectText)
	body := getBody(msg)

	auth := smtp.PlainAuth("",
		config.Config.SMTPUser,
		config.Config.SMTPPass,
		config.Config.SMTPHost)
	if err := mailer.SendMail(config.Config.SMTPHost+":"+config.Config.SMTPPort,
		auth,
		config.Config.SMTPUser,
		toList,
		[]byte(header+body)); err != nil {
		log.Println(err.Error())
		return errors.New("cannot send email to " + to)
	}
	return nil
}
