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
package service

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/batk0/gc-tracker/config"
	"github.com/batk0/gc-tracker/data"
	"github.com/gorilla/sessions"
)

type GCTrackerService struct {
	session *sessions.Session
	data    data.GCTrackerData
}

func NewGCTrackerService(d data.GCTrackerData) *GCTrackerService {
	if d == nil {
		d = &data.FirestoreGCTrackerData{}
	}
	return &GCTrackerService{data: d}
}

func (s *GCTrackerService) RenderPage(content, errorMsg string) string {
	str := header
	str += s.RenderError(errorMsg)
	str += content
	str += footer
	return str
}

func (s *GCTrackerService) ShowStyle() string {
	// TODO: Test?
	return showStyle()
}

func (s *GCTrackerService) GetResetToken() string {
	// TODO: Test it
	if s.session == nil {
		return ""
	}
	return fmt.Sprint(s.session.Values["resetToken"])
}

func (s *GCTrackerService) SetResetToken(token string) {
	s.session.Values["resetToken"] = token
	// TODO: Save session and test
}

func (s *GCTrackerService) SignIn(form url.Values) error {
	user := s.data.NewUser()

	user.Set(form)
	if err := user.Authenticate(); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func (s *GCTrackerService) UpdateCases() error {
	for _, c := range s.data.GetAllCases() {
		if err := c.CheckStatus(); err == nil {
			continue
		} else if err.Error() == "status changed" {
			c.Create()
		} else {
			return err
		}
	}
	return nil
}

func (s *GCTrackerService) GetSession(r *http.Request) {
	store := s.data.NewSession()

	if store == nil {
		log.Println("Session store does not exist")
	}

	session, err := store.Get(r, config.Config.Cookie)
	if err != nil {
		log.Println("Cannot get session " + err.Error())
	} else {
		s.session = session
	}
}

func (s *GCTrackerService) IsAuthenticated() bool {
	if s.session == nil {
		log.Println("IsAuthenticated(): session is nil")
		return false
	}
	return s.session.Values["authenticated"] != nil && s.session.Values["authenticated"].(bool) && s.session.Values["username"] != nil
}

func (s *GCTrackerService) SetAuthenticated(auth bool) {
	s.session.Values["authenticated"] = auth
	// TODO: Save session and test it
}

func (s *GCTrackerService) SignUp(formData url.Values) error {
	user := s.data.NewUser()

	user.Set(formData)
	if err := user.Validate(true); err != nil {
		log.Println(err.Error())
		return err
	}
	if err := user.HashAndSalt(); err != nil {
		log.Println(err.Error())
		return err
	}
	if err := s.data.CreateUser(user); err != nil {
		log.Println(err.Error())
		return err
	}
	user.SendNotification("Your account '" + user.GetUsername() + "' has been created.")
	return nil
}

func (s *GCTrackerService) ResetPwd(r *http.Request) error {
	if username := r.PostForm.Get("username"); username != "" {
		if s.data.UserAvailable(username) {
			return errors.New("user not found")
		}
		user := s.data.NewUser()
		user.GetByUsername(username)
		address := r.URL.Scheme + r.URL.Host + "/changepwd"
		if err := user.GenerateResetToken(address); err != nil {
			return errors.New("cannot generate reset token")
		}
	} else {
		return errors.New("username is not specified")
	}
	return nil
}

func (s *GCTrackerService) ChangePwd(r *http.Request) error {
	var user data.GCTrackerUser
	username := fmt.Sprint(s.session.Values["username"])
	if s.IsAuthenticated() {
		user = s.data.NewUser()
		if err := user.GetByUsername(username); err != nil {
			log.Println(err.Error())
			return errors.New("cannot find user")
		}
	} else {
		var err error
		token := fmt.Sprint(s.session.Values["resetToken"])
		user, err = s.data.GetUserByResetToken(token)
		if err != nil {
			return err
		}
	}
	user.SetPassword2(r.PostForm.Get("password"), r.PostForm.Get("password2"))
	if err := user.Validate(false); err != nil {
		log.Println(err.Error())
		return err
	}
	if err := user.HashAndSalt(); err != nil {
		log.Println(err.Error())
		return err
	}
	if err := user.Update(); err != nil {
		log.Println(err.Error())
		return err
	}
	user.SendNotification("Your password has been changed.")
	return nil
}

func (s *GCTrackerService) ShowUsers() string {
	str := ""
	errorMsg := ""
	// TODO: Re-implement in secure way
	// if users, err := data.GetUsers(); err == nil {
	// 	for _, doc := range users {
	// 		if err == iterator.Done {
	// 			break
	// 		}
	// 		if err != nil {
	// 			log.Println("Cannot iterate: " + err.Error())
	// 			errorMsg += err.Error()
	// 			break
	// 		}
	// 		log.Println(doc.Data())
	// 		str += fmt.Sprintln(doc.Data())
	// 	}
	// } else {
	// 	errorMsg += err.Error()
	// }

	return s.RenderPage(str, errorMsg)
}

func (s *GCTrackerService) RenderCases() string {
	user := s.data.NewUser()
	username := fmt.Sprint(s.session.Values["username"])
	user.GetByUsername(username)

	cases := user.GetCases()
	if cases == nil {
		return ""
	}
	str := ""
	for _, c := range cases {
		str += `<tr><td class=check><input type=checkbox name=cases value=`
		str += c.GetID()
		str += `></td><td>`
		str += c.GetID()
		str += `</td><td>`
		str += c.GetName()
		str += `</td><td>`
		str += c.GetStatus()
		str += `</td></tr>`
	}
	return str
}

func (s *GCTrackerService) AddCase(formData url.Values) {
	user := s.data.NewUser()
	username := fmt.Sprint(s.session.Values["username"])
	if err := user.GetByUsername(username); err == nil {
		c := s.data.NewCase()
		c.Set(formData)
		c.CheckStatus()
		if err := user.AddCase(c); err != nil {
			log.Println(err.Error())
			return
		}
		log.Println("Add case " + c.GetID())
		if err := user.Update(); err != nil {
			log.Println(err.Error())
			return
		}
	}
}

func (s *GCTrackerService) DelCases(cases []string) {
	user := s.data.NewUser()
	username := fmt.Sprint(s.session.Values["username"])
	if err := user.GetByUsername(username); err == nil {
		for _, c := range cases {
			log.Println("Delete case " + c)
			user.DelCase(c)
			if err := user.Update(); err != nil {
				log.Println(err.Error())
				return
			}
		}
	}
}

func (s *GCTrackerService) RenderError(errorMsg string) string {
	if errorMsg != "" {
		str := "<div class='error'><ul>"
		for _, s := range strings.Split(errorMsg, "\n") {
			if s != "" {
				str += "<li>" + s
			}
		}
		str += "</ul></div>"
		return str
	}
	return ""
}
