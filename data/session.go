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
	"log"

	fsgsession "github.com/GoogleCloudPlatform/firestore-gorilla-sessions"
	"github.com/gorilla/sessions"
)

func (d *FirestoreGCTrackerData) NewSession() sessions.Store {
	ctx := context.Background()
	client := d.connectFirestore(ctx)

	store, err := fsgsession.New(ctx, client)
	if err != nil {
		log.Println("Cannot create session store " + err.Error())
		return nil
	}

	return store
}
