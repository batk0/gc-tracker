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
	"context"
	"errors"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/batk0/gc-tracker/config"
	"github.com/gorilla/sessions"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GCTrackerData interface {
	NewSession() sessions.Store
	NewUser() GCTrackerUser
	UserAvailable(string) bool
	GetUser(string) (*firestore.DocumentSnapshot, error)
	UpdateUser(user GCTrackerUser) error
	GetUserByResetToken(string) (GCTrackerUser, error)
	GetUsersByCase(string) []GCTrackerUser
	CreateUser(GCTrackerUser) error
	NewCase() GCTrackerCase
	GetCase(string) (*firestore.DocumentSnapshot, error)
	GetCases([]string) []GCTrackerCase
	GetAllCases() []GCTrackerCase
	CreateCase(GCTrackerCase) error
	DeleteCase(GCTrackerCase) error
}

type FirestoreGCTrackerData struct{}

func (d *FirestoreGCTrackerData) connectFirestore(ctx context.Context) *firestore.Client {
	projectID := config.Config.Project
	if config.Config.IsAppEngine {
		client, err := firestore.NewClient(ctx, projectID)
		if err != nil {
			log.Fatalln("Cannot connect to Firestore: " + err.Error())
		}
		return client
	}
	sa := option.WithCredentialsFile(".sa-key.json")
	client, err := firestore.NewClient(ctx, projectID, sa)
	if err != nil {
		log.Fatalln("Cannot connect to Firestore: " + err.Error())
	}
	return client
}

func (d *FirestoreGCTrackerData) CreateUser(user GCTrackerUser) error {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	userDoc := client.Doc("users/" + user.GetUsername())
	if _, err := userDoc.Create(ctx, user); err != nil {
		log.Println("Cannot create user: " + err.Error())
		return err
	}
	return nil
}

func (d *FirestoreGCTrackerData) GetUsers() ([]*firestore.DocumentSnapshot, error) {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	return client.Collection("users").Documents(ctx).GetAll()
}

func (d *FirestoreGCTrackerData) GetUser(username string) (*firestore.DocumentSnapshot, error) {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	user := client.Doc("users/" + username)

	return user.Get(ctx)
}

func (d *FirestoreGCTrackerData) GetUsersByCase(id string) []GCTrackerUser {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	usersRef := client.Collection("users")

	q := usersRef.Where("cases."+id, "==", true)
	iter := q.Documents(ctx)
	defer iter.Stop()
	var users []GCTrackerUser
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println(err.Error())
		} else {
			u := d.NewUser()
			doc.DataTo(u)
			users = append(users, u)
		}
	}
	return users
}

func (d *FirestoreGCTrackerData) GetUserByResetToken(token string) (GCTrackerUser, error) {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	usersRef := client.Collection("users")

	q := usersRef.Where("reset.Token", "==", token).
		Where("reset.Timestamp", ">", time.Now().Unix()-3600)
	iter := q.Documents(ctx)
	defer iter.Stop()
	u := d.NewUser()
	doc, err := iter.Next()
	if err != iterator.Done && err == nil {
		doc.DataTo(u)
		return u, nil
	}
	log.Println(err.Error())
	return u, errors.New("token not found")
}

func (d *FirestoreGCTrackerData) UserAvailable(username string) bool {
	log.Println("Checking user: " + username)
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	user := client.Doc("users/" + username)

	if _, err := user.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return true
		} else {
			log.Println(err.Error())
		}
	}
	return false
}

func (d *FirestoreGCTrackerData) UpdateUser(user GCTrackerUser) error {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	userDoc := client.Doc("users/" + user.GetUsername())
	if _, err := userDoc.Set(ctx, user); err != nil {
		log.Println("Cannot update user: " + err.Error())
		return err
	}
	return nil
}

func (d *FirestoreGCTrackerData) CreateCase(c GCTrackerCase) error {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	caseDoc := client.Doc("cases/" + c.GetID())
	if _, err := caseDoc.Set(ctx, c); err != nil {
		log.Println("Cannot create case: " + err.Error())
		return err
	}
	return nil
}

func (d *FirestoreGCTrackerData) DeleteCase(c GCTrackerCase) error {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	caseDoc := client.Doc("cases/" + c.GetID())
	if _, err := caseDoc.Delete(ctx); err != nil {
		log.Println("Cannot delete case: " + err.Error())
		return err
	}
	return nil
}

func (d *FirestoreGCTrackerData) GetCase(c string) (*firestore.DocumentSnapshot, error) {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	caseRef := client.Doc("cases/" + c)

	return caseRef.Get(ctx)
}

func (d *FirestoreGCTrackerData) GetCases(ids []string) []GCTrackerCase {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	casesRef := client.Collection("cases")

	q := casesRef.Where("id", "in", ids)
	iter := q.Documents(ctx)
	defer iter.Stop()
	var cases []GCTrackerCase
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println(err.Error())
		} else {
			c := d.NewCase()
			doc.DataTo(c)
			cases = append(cases, c)
		}
	}
	return cases
}

func (d *FirestoreGCTrackerData) GetAllCases() []GCTrackerCase {
	ctx := context.Background()
	client := d.connectFirestore(ctx)
	defer client.Close()

	iter := client.Collection("cases").Documents(ctx)
	defer iter.Stop()
	var cases []GCTrackerCase
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println(err.Error())
		} else {
			c := d.NewCase()
			doc.DataTo(c)
			cases = append(cases, c)
		}
	}
	return cases
}
