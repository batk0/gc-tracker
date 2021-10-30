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
	"log"
	"net/smtp"

	"github.com/batk0/gc-tracker/config"
)

func Send(to string, msg string) error {
	toList := []string{to}
	header := "From: " + config.Config.SMTPUser
	header += "\nTo: " + to
	header += `
Subject: Notification from GC-Tracker

Hello!

`
	footer := `
This is automated message. Please do not reply.
`
	auth := smtp.PlainAuth("",
		config.Config.SMTPUser,
		config.Config.SMTPPass,
		config.Config.SMTPHost)
	if err := smtp.SendMail(config.Config.SMTPHost+":"+config.Config.SMTPPort,
		auth,
		config.Config.SMTPUser,
		toList,
		[]byte(header+msg+footer)); err != nil {
		log.Println(err.Error())
		return errors.New("cannot send email to " + to)
	}
	return nil
}
