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
	"net/smtp"
	"testing"

	"github.com/batk0/gc-tracker/config"
)

type spyMailer struct {
	addr string
	auth smtp.Auth
	from string
	to   []string
	msg  []byte
}

func (m *spyMailer) SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	m.addr = addr
	m.auth = a
	m.from = from
	m.to = to
	m.msg = msg
	if m.addr == ":" {
		return errors.New("wrong smtp address")
	}
	if m.from == "" {
		return errors.New("wrong smtp user")
	}
	if len(m.to) == 0 {
		return errors.New("empty recepients list")
	}
	if len(m.msg) == 0 {
		return errors.New("empty message")
	}
	return nil
}

func (m *spyMailer) String() string {
	return fmt.Sprintf("addr: %s, auth: %s, from: %s, to: %s, msg: %s", m.addr, m.auth, m.from, m.to, m.msg)
}

func Test_getHeaders(t *testing.T) {
	type args struct {
		from    string
		to      string
		subject string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			name: "Correct values",
			args: args{"from@example.com", "to@example.com", "subj"},
			want: `From: from@example.com
To: to@example.com
Subject: subj

`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHeaders(tt.args.from, tt.args.to, tt.args.subject); got != tt.want {
				t.Errorf("getHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getBody(t *testing.T) {
	type args struct {
		msg string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			"Some message",
			args{"Message"},
			`Hello!

Message

This is automated message. Please do not reply.
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBody(tt.args.msg); got != tt.want {
				t.Errorf("getBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sendReal(t *testing.T) {
	type args struct {
		to  string
		msg string
	}
	type cfg struct {
		SMTPHost string
		SMTPPort string
		SMTPUser string
		SMTPPass string
	}
	tests := []struct {
		name    string
		args    args
		cfg     cfg
		want    spyMailer
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "Positive: Send Mail",
			args: args{"to@example.com", "Message"},
			cfg:  cfg{"smtp.example.com", "587", "from@example.com", "password"},
			want: spyMailer{
				"smtp.example.com:587",
				smtp.PlainAuth("", "from@example.com", "password", "smtp.example.com"),
				"from@example.com",
				[]string{"to@example.com"},
				([]byte)(getHeaders("from@example.com", "to@example.com", subjectText) + getBody("Message")),
			},
			wantErr: false,
		},
		{
			name: "Negative: Send Mail incorrect To",
			args: args{"", "Message"},
			cfg:  cfg{"smtp.example.com", "587", "from@example.com", "password"},
			want: spyMailer{
				"smtp.example.com:587",
				smtp.PlainAuth("", "from@example.com", "password", "smtp.example.com"),
				"from@example.com",
				[]string{""},
				([]byte)(getHeaders("from@example.com", "", subjectText) + getBody("Message")),
			},
			wantErr: true,
		},
		{
			name: "Negative: Send Mail incorrect From",
			args: args{"to@example.com", "Message"},
			cfg:  cfg{"smtp.example.com", "587", "", "password"},
			want: spyMailer{
				"smtp.example.com:587",
				smtp.PlainAuth("", "", "password", "smtp.example.com"),
				"",
				[]string{"to@example.com"},
				([]byte)(getHeaders("", "to@example.com", subjectText) + getBody("Message")),
			},
			wantErr: true,
		},
		{
			name: "Negative: Send Mail incorrect Message",
			args: args{"to@example.com", ""},
			cfg:  cfg{"smtp.example.com", "587", "from@example.com", "password"},
			want: spyMailer{
				"smtp.example.com:587",
				smtp.PlainAuth("", "from@example.com", "password", "smtp.example.com"),
				"from@example.com",
				[]string{"to@example.com"},
				([]byte)(getHeaders("from@example.com", "to@example.com", subjectText) + getBody("")),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		mailer := new(spyMailer)
		config.Config.SMTPHost = tt.cfg.SMTPHost
		config.Config.SMTPPort = tt.cfg.SMTPPort
		config.Config.SMTPUser = tt.cfg.SMTPUser
		config.Config.SMTPPass = tt.cfg.SMTPPass
		t.Run(tt.name, func(t *testing.T) {
			if err := sendReal(mailer, tt.args.to, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("sendReal() error = %v, wantErr = %v", err, tt.wantErr)
			} else if err == nil && mailer.String() != tt.want.String() {
				t.Errorf("sendReal() got = %v, want = %v", mailer.String(), tt.want.String())
			}

		})
	}
}
