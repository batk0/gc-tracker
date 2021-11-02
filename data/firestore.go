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
	fsgsession "github.com/GoogleCloudPlatform/firestore-gorilla-sessions"
	"github.com/batk0/gc-tracker/config"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func connectFirestore(ctx context.Context) *firestore.Client {
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

func CreateUser(user User) error {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	userDoc := client.Doc("users/" + user.Username)
	if _, err := userDoc.Create(ctx, user); err != nil {
		log.Println("Cannot create user: " + err.Error())
		return err
	}
	return nil
}

func GetUsers() ([]*firestore.DocumentSnapshot, error) {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	return client.Collection("users").Documents(ctx).GetAll()
}

func GetUser(username string) (*firestore.DocumentSnapshot, error) {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	user := client.Doc("users/" + username)

	return user.Get(ctx)
}

func GetUsersByCase(id string) []User {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	usersRef := client.Collection("users")

	q := usersRef.Where("cases."+id, "==", true)
	iter := q.Documents(ctx)
	defer iter.Stop()
	var users []User
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println(err.Error())
		} else {
			var u User
			doc.DataTo(&u)
			users = append(users, u)
		}
	}
	return users
}

func GetUserByResetToken(token string) (User, error) {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	usersRef := client.Collection("users")

	q := usersRef.Where("reset.Token", "==", token).
		Where("reset.Timestamp", ">", time.Now().Unix()-3600)
	iter := q.Documents(ctx)
	defer iter.Stop()
	var u User
	doc, err := iter.Next()
	if err != iterator.Done && err == nil {
		doc.DataTo(&u)
		return u, nil
	}
	log.Println(err.Error())
	return u, errors.New("token not found")
}

func UserAvailable(username string) bool {
	log.Println("Checking user: " + username)
	ctx := context.Background()
	client := connectFirestore(ctx)
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

func NewSession() *fsgsession.Store {
	ctx := context.Background()
	client := connectFirestore(ctx)

	store, err := fsgsession.New(ctx, client)
	if err != nil {
		log.Println("Cannot create session store " + err.Error())
		return nil
	}

	return store
}

func UpdateUser(user User) error {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	userDoc := client.Doc("users/" + user.Username)
	if _, err := userDoc.Set(ctx, user); err != nil {
		log.Println("Cannot update user: " + err.Error())
		return err
	}
	return nil
}

func CreateCase(c Case) error {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	caseDoc := client.Doc("cases/" + c.ID)
	if _, err := caseDoc.Set(ctx, c); err != nil {
		log.Println("Cannot create case: " + err.Error())
		return err
	}
	return nil
}

func DeleteCase(c Case) error {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	caseDoc := client.Doc("cases/" + c.ID)
	if _, err := caseDoc.Delete(ctx); err != nil {
		log.Println("Cannot delete case: " + err.Error())
		return err
	}
	return nil
}

func GetCase(c string) (*firestore.DocumentSnapshot, error) {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	caseRef := client.Doc("cases/" + c)

	return caseRef.Get(ctx)
}

func GetCases(ids []string) []Case {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	casesRef := client.Collection("cases")

	q := casesRef.Where("id", "in", ids)
	iter := q.Documents(ctx)
	defer iter.Stop()
	var cases []Case
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println(err.Error())
		} else {
			var c Case
			doc.DataTo(&c)
			cases = append(cases, c)
		}
	}
	return cases
}

func GetAllCases() []Case {
	ctx := context.Background()
	client := connectFirestore(ctx)
	defer client.Close()

	iter := client.Collection("cases").Documents(ctx)
	defer iter.Stop()
	var cases []Case
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println(err.Error())
		} else {
			var c Case
			doc.DataTo(&c)
			cases = append(cases, c)
		}
	}
	return cases
}
