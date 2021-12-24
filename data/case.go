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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/schema"
	"gopkg.in/go-playground/validator.v9"
)

type GCTrackerCase interface {
	CheckStatus() error
	Create()
	Set(url.Values)
	GetID() string
	GetName() string
	GetStatus() string
	Validate() error
}

type GCTrackerCaseImpl struct {
	ID        string        `firestore:"id" schema:"case" validate:"alphanum,len=13"`
	Name      string        `firestore:"name" schema:"name" validate:"alphanum,min=0,max=40"`
	Status    string        `firestore:"status" schema:"-"`
	OldStatus string        `firestore:"old" schema:"-"`
	data      GCTrackerData `firestore:"-" schema:"-"`
}

func (d *FirestoreGCTrackerData) NewCase() GCTrackerCase { return &GCTrackerCaseImpl{data: d} }
func (c *GCTrackerCaseImpl) Create()                     { c.data.CreateCase(c) }
func (c *GCTrackerCaseImpl) Delete()                     { c.data.DeleteCase(c) }
func (c *GCTrackerCaseImpl) GetID() string               { return c.ID }
func (c *GCTrackerCaseImpl) GetName() string             { return c.Name }
func (c *GCTrackerCaseImpl) GetStatus() string           { return c.Status }

func (c *GCTrackerCaseImpl) Validate() error {
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

func (c *GCTrackerCaseImpl) GetByID(id string) error {
	caseSnap, err := c.data.GetCase(id)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Println("GCTrackerCase does not exist: " + err.Error())
			return errors.New("case does not exist")
		}
		return err
	}

	if err := caseSnap.DataTo(c); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (c *GCTrackerCaseImpl) Set(formData url.Values) {
	decoder := schema.NewDecoder()
	if err := decoder.Decode(c, formData); err != nil {
		log.Println(err.Error())
	}
}

func (c *GCTrackerCaseImpl) CheckStatus() error {
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
	if resp.StatusCode != http.StatusOK {
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

	if c.Status != status {
		log.Println(c.ID + " case status changed")
		for _, user := range c.data.GetUsersByCase(c.ID) {
			user.SendNotification("Your case " + c.Name + " status has changed")
		}
		c.OldStatus = c.Status
		c.Status = status
		return errors.New("status changed")
	}
	return nil
}
