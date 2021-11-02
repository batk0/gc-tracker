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
package main

import (
	"log"
	"net/http"

	"github.com/batk0/gc-tracker/config"
	"github.com/batk0/gc-tracker/handlers"
)

func main() {

	if err := config.InitConfig(); err != nil {
		log.Fatalln(err.Error())
	}

	http.HandleFunc("/", handlers.IndexHandler)
	http.HandleFunc("/resetpwd", handlers.ResetPwdHandler)
	http.HandleFunc("/changepwd", handlers.ChangePwdHandler)
	http.HandleFunc("/signup", handlers.SignUpHandler)
	http.HandleFunc("/signin", handlers.SignInHandler)
	http.HandleFunc("/signout", handlers.SignOutHandler)
	http.HandleFunc("/case", handlers.CaseHandler)
	http.HandleFunc("/update", handlers.UpdateHandler)
	http.HandleFunc("/users", handlers.UsersHandler)
	http.HandleFunc("/style.css", handlers.StyleHandler)
	log.Printf("Listening at %s", config.Config.Port)

	if err := http.ListenAndServe(":"+config.Config.Port, nil); err != nil {
		log.Fatalf("FATAL: %s", err)
	}
}
