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
	"log"
	"net/http"

	"github.com/batk0/gc-tracker/config"
	"github.com/gorilla/sessions"
)

func GetSession(r *http.Request) *sessions.Session {
	store := NewSession()

	if store == nil {
		log.Println("Session store does not exist")
	}

	session, err := store.Get(r, config.Config.Cookie)
	if err != nil {
		log.Println("Cannot get session " + err.Error())
		return nil
	}

	return session
}
