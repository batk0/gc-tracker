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
package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/batk0/gc-tracker/data"
	"github.com/gorilla/sessions"
	"google.golang.org/api/iterator"
)

func signIn(formData url.Values) error {
	user := new(data.User)

	user.Set(formData)
	if err := user.Authenticate(); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func signUp(formData url.Values) error {
	user := new(data.User)

	user.Set(formData)
	if err := user.Validate(true); err != nil {
		log.Println(err.Error())
		return err
	}
	if err := user.HashAndSalt(); err != nil {
		log.Println(err.Error())
		return err
	}
	if err := data.CreateUser(*user); err != nil {
		log.Println(err.Error())
		return err
	}
	user.SendNotfication("Your account '" + user.Username + "' has been created.")
	return nil
}

func resetPwd(r *http.Request) error {
	if username := r.PostForm.Get("username"); username != "" {
		if data.UserAvailable(username) {
			return errors.New("user not found")
		}
		user := data.User{Username: username}
		user.Get()
		address := r.URL.Scheme + r.URL.Host + "/changepwd"
		if err := user.GenerateResetToken(address); err != nil {
			return errors.New("cannot generate reset token")
		}
	} else {
		return errors.New("username is not specified")
	}
	return nil
}

func changePwd(s sessions.Session, r *http.Request) error {
	var user data.User
	if isAuthenticated(s) {
		user.Username = fmt.Sprint(s.Values["username"])
		if err := user.Get(); err != nil {
			log.Println(err.Error())
			return errors.New("cannot find user")
		}
	} else {
		var err error
		user, err = data.GetUserByResetToken(fmt.Sprint(s.Values["resetToken"]))
		if err != nil {
			return err
		}
	}
	user.Password = r.PostForm.Get("password")
	user.ConfirmPassword = r.PostForm.Get("password2")
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
	user.SendNotfication("Your password has been changed.")
	return nil
}

func showUsers() string {
	str := ""
	errorMsg := ""

	if users, err := data.GetUsers(); err == nil {
		for _, doc := range users {
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Println("Cannot iterate: " + err.Error())
				errorMsg += err.Error()
				break
			}
			log.Println(doc.Data())
			str += fmt.Sprintln(doc.Data())
		}
	} else {
		errorMsg += err.Error()
	}

	return renderPage(str, errorMsg)
}

func isAuthenticated(s sessions.Session) bool {
	return s.Values["authenticated"] != nil && s.Values["authenticated"].(bool) && s.Values["username"] != nil
}

func renderCases(s sessions.Session) string {
	user := data.User{
		Username: fmt.Sprint(s.Values["username"]),
	}
	user.Get()

	cases := user.GetCases()
	if cases == nil {
		return ""
	}
	str := ""
	for _, c := range cases {
		str += `<tr><td class=check><input type=checkbox name=cases value=`
		str += c.ID
		str += `></td><td>`
		str += c.ID
		str += `</td><td>`
		str += c.Name
		str += `</td><td>`
		str += c.Status
		str += `</td></tr>`
	}
	return str
}

func addCase(s sessions.Session, formData url.Values) {
	user := new(data.User)
	user.Username = fmt.Sprint(s.Values["username"])
	if err := user.Get(); err == nil {
		c := new(data.Case)
		c.Set(formData)
		c.CheckStatus()
		if err := user.AddCase(*c); err != nil {
			log.Println(err.Error())
			return
		}
		log.Println("Add case " + c.ID)
		if err := user.Update(); err != nil {
			log.Println(err.Error())
			return
		}
	}
}

func updateCases() error {
	for _, c := range data.GetAllCases() {
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

func delCase(s sessions.Session, cases []string) {
	user := data.User{
		Username: fmt.Sprint(s.Values["username"]),
	}
	if err := user.Get(); err == nil {
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
