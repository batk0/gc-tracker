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
	"net/url"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/batk0/gc-tracker/mailer"
	"github.com/google/uuid"
	"github.com/gorilla/schema"
	"gopkg.in/go-playground/validator.v9"
)

type GCTrackerUser interface {
	Set(url.Values)
	Authenticate() error
	Validate(bool) error
	HashAndSalt() error
	SendNotification(string)
	GetUsername() string
	GetByUsername(string) error
	GenerateResetToken(string) error
	SetPassword2(string, string)
	Update() error
	AddCase(GCTrackerCase) error
	DelCase(string)
	GetCases() []GCTrackerCase
}

type resetPassword struct {
	Token     string
	Timestamp int64
}

type GCTrackerUserImpl struct {
	Username        string          `firestore:"username" schema:"username" validate:"required,min=2,max=80,alphanum,available"`
	Email           string          `firestore:"email" schema:"email" validate:"required,email"`
	Password        string          `firestore:"password" schema:"password" validate:"required,min=8,max=80,eqfield=ConfirmPassword"`
	ConfirmPassword string          `firestore:"-" schema:"password2"`
	Cases           map[string]bool `firestore:"cases" schema:"-"`
	Reset           resetPassword   `firestore:"reset" schema:"-"`
	data            GCTrackerData   `firestore:"-" schema:"-"`
}

func (d *FirestoreGCTrackerData) NewUser() GCTrackerUser { return &GCTrackerUserImpl{data: d} }
func (u *GCTrackerUserImpl) GetUsername() string         { return u.Username }

func (u *GCTrackerUserImpl) Set(formData url.Values) {
	decoder := schema.NewDecoder()
	if err := decoder.Decode(u, formData); err != nil {
		log.Println(err.Error())
	}
}

func (u *GCTrackerUserImpl) SetPassword2(p1, p2 string) {
	u.Password = p1
	u.ConfirmPassword = p2
}

func (u *GCTrackerUserImpl) Validate(checkAvailable bool) error {
	v := validator.New()
	if err := v.RegisterValidation("available", func(fl validator.FieldLevel) bool {
		return !checkAvailable || u.data.UserAvailable(fl.Field().String())
	}); err != nil {
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

func (u *GCTrackerUserImpl) HashAndSalt() error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.MinCost)
	if err == nil {
		u.Password = string(hash)
	}

	return err
}

func (u *GCTrackerUserImpl) GetByUsername(username string) error {
	userSnap, err := u.data.GetUser(username)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Println("GCTrackerUser does not exist: " + err.Error())
			return errors.New("user does not exist")
		}
		return err
	}
	if err := userSnap.DataTo(u); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (u *GCTrackerUserImpl) GenerateResetToken(url string) error {
	token := uuid.New()
	u.Reset.Token = token.String()

	u.Reset.Timestamp = time.Now().Unix()
	if err := u.Update(); err != nil {
		log.Println("Cannot update user " + u.Username)
		return errors.New("cannot save reset token")
	}
	address := url + "?a=r&t=" + token.String()
	defer u.SendNotification("Please follow the link " + address + " to reset your password.")
	return nil
}

func (u *GCTrackerUserImpl) Authenticate() error {
	userSnap, err := u.data.GetUser(u.Username)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Println("GCTrackerUser does not exist: " + err.Error())
			return errors.New("user does not exist")
		}
		return err
	}

	var dbUser *GCTrackerUserImpl
	if err := userSnap.DataTo(dbUser); err != nil {
		log.Println(err.Error())
		return err
	}
	if u.Username != dbUser.Username {
		return errors.New("username does not match")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(u.Password)); err != nil {
		log.Println(err.Error())
		return errors.New("password does not match")
	}
	return nil

}

func (u *GCTrackerUserImpl) AddCase(c GCTrackerCase) error {
	u.Cases[c.GetID()] = true
	if err := c.Validate(); err != nil {
		return err
	}
	c.Create()
	return nil
}

func (u *GCTrackerUserImpl) DelCase(c string) {
	delete(u.Cases, c)
	(&GCTrackerCaseImpl{ID: c}).Delete()
}

func (u *GCTrackerUserImpl) Update() error {
	u.data.UpdateUser(u)
	return nil
}

func (u *GCTrackerUserImpl) GetCases() []GCTrackerCase {
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
	gotCases := u.data.GetCases(cases)
	// retCases := make([]GCTrackerCase, len(gotCases))
	// for i, c := range gotCases {
	// 	// c.data = u.data
	// 	retCases[i] = GCTrackerCase(c)
	// }
	return gotCases
}

func (u *GCTrackerUserImpl) SendNotification(msg string) {
	if err := mailer.Send(u.Email, msg); err != nil {
		log.Println("Cannot send email: " + err.Error())
	}
}
