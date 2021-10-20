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
package data

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/schema"
	"gopkg.in/go-playground/validator.v9"
)

type User struct {
	Username        string          `firestore:"username" schema:"username" validate:"required,min=2,max=80,alphanum,available"`
	Email           string          `firestore:"email" schema:"email" validate:"required,email"`
	Password        string          `firestore:"password" schema:"password" validate:"required,min=8,max=80,eqfield=ConfirmPassword"`
	ConfirmPassword string          `firestore:"-" schema:"password2"`
	Cases           map[string]bool `firestore:"cases" schema:"-"`
}

func (u *User) Set(formData url.Values) {
	decoder := schema.NewDecoder()
	if err := decoder.Decode(u, formData); err != nil {
		log.Println(err.Error())
	}
}

func (u User) Validate(checkAvailable bool) error {
	v := validator.New()
	if err := v.RegisterValidation("available", func(fl validator.FieldLevel) bool { return !checkAvailable || UserAvailable(fl.Field().String()) }); err != nil {
		log.Println(err.Error())
	}

	if err := v.Struct(u); err != nil {
		errorMsg := ""
		for _, e := range err.(validator.ValidationErrors) {
			log.Println(e)
			errorMsg += fmt.Sprintln(e)
		}
		return errors.New(errorMsg)
	}
	return nil
}

func (u *User) HashAndSalt() error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.MinCost)
	if err == nil {
		u.Password = string(hash)
	}

	return err
}

func (user *User) Get() error {
	userSnap, err := GetUser(user.Username)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Println("User does not exist: " + err.Error())
			return errors.New("user does not exist")
		}
		return err
	}

	if err := userSnap.DataTo(&user); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (user User) Authenticate() error {
	userSnap, err := GetUser(user.Username)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Println("User does not exist: " + err.Error())
			return errors.New("user does not exist")
		}
		return err
	}

	var dbUser User
	if err := userSnap.DataTo(&dbUser); err != nil {
		log.Println(err.Error())
		return err
	}
	if user.Username != dbUser.Username {
		return errors.New("username does not match")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password)); err != nil {
		log.Println(err.Error())
		return errors.New("password does not match")
	}
	return nil

}

func (u *User) AddCase(c Case) error {
	u.Cases[c.ID] = true
	if err := c.Validate(); err != nil {
		return err
	}
	c.Create()
	return nil
}

func (u *User) DelCase(c string) {
	delete(u.Cases, c)
	Case{ID: c}.Delete()
}

func (u User) Update() error {
	UpdateUser(u)
	return nil
}

func (u User) GetCases() []Case {
	l := len(u.Cases)
	if l == 0 {
		return nil
	}
	cases := make([]string, l)
	i := 0
	for c := range u.Cases {
		cases[i] = c
		i++
	}
	log.Println(cases)
	return GetCases(cases)
}

type Case struct {
	ID        string `firestore:"id" schema:"case" validate:"alphanum,len=13"`
	Name      string `firestore:"name" schema:"name" validate:"alphanum,min=0,max=40"`
	Status    string `firestore:"status" schema:"-"`
	OldStatus string `firestore:"old" schema:"-"`
}

func (c Case) Validate() error {
	v := validator.New()

	if err := v.Struct(c); err != nil {
		errorMsg := ""
		for _, e := range err.(validator.ValidationErrors) {
			log.Println(e)
			errorMsg += fmt.Sprintln(e)
		}
		return errors.New(errorMsg)
	}
	return nil
}

func (c *Case) Get() error {
	caseSnap, err := GetCase(c.ID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Println("Case does not exist: " + err.Error())
			return errors.New("case does not exist")
		}
		return err
	}

	if err := caseSnap.DataTo(&c); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (c Case) Create() {
	CreateCase(c)
}

func (c Case) Delete() {
	DeleteCase(c)
}

func (c *Case) Set(formData url.Values) {
	decoder := schema.NewDecoder()
	if err := decoder.Decode(c, formData); err != nil {
		log.Println(err.Error())
	}
}

func (c *Case) CheckStatus() error {
	form := url.Values{
		"completedActionsCurrentPage": []string{"0"},
		"upcomingActionsCurrentPage":  []string{"0"},
		"appReceiptNum":               []string{c.ID},
		"caseStatusSearchBtn":         []string{"CHECK+STATUS"},
	}
	resp, err := http.PostForm("https://egov.uscis.gov/casestatus/mycasestatus.do", form)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("non-ok response received")
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}
	doc.Find(".current-status-sec strong").Remove()
	doc.Find(".current-status-sec span").Remove()
	status := strings.TrimSpace(doc.Find(".current-status-sec").Text())

	log.Println(status)

	if c.OldStatus != status {
		// TODO notifications
		c.OldStatus = c.Status
		c.Status = status
	}
	return nil
}
