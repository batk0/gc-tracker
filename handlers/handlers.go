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
	"fmt"
	"log"
	"net/http"

	"github.com/batk0/gc-tracker/data"
)

func CaseHandler(w http.ResponseWriter, r *http.Request) {
	session := data.GetSession(r)
	if isAuthenticated(*session) && r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			log.Println(err.Error())
		} else if r.PostForm.Get("add") != "" {
			addCase(*session, r.PostForm)
		} else if r.PostForm.Get("delete") != "" {
			log.Println(r.PostForm)
			delCase(*session, r.PostForm["cases"])
		}
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(303)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	session := data.GetSession(r)
	if isAuthenticated(*session) {
		fmt.Fprint(w, showCases(*session))
	} else {
		w.Header().Set("Location", "/signin")
		w.WriteHeader(303)
		fmt.Fprint(w, renderPage("Please SignIn first", ""))
	}
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	session := data.GetSession(r)
	if isAuthenticated(*session) {
		fmt.Fprint(w, showUsers())
	} else {
		w.Header().Set("Location", "/signin")
		w.WriteHeader(303)
	}
}

func ResetPwdHandler(w http.ResponseWriter, r *http.Request) {
	// TODO reset password
	fmt.Fprint(w, showResetPwd())
}

func SignInHandler(w http.ResponseWriter, r *http.Request) {
	session := data.GetSession(r)
	if isAuthenticated(*session) {
		w.Header().Set("Location", "/")
		w.WriteHeader(303)
		fmt.Fprint(w, renderPage("Already signed in", ""))
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			fmt.Fprint(w, showSignIn(err.Error()))
		} else if err := signIn(r.PostForm); err != nil {
			fmt.Fprint(w, showSignIn(err.Error()))
		} else {
			session.Values["authenticated"] = true
			session.Values["username"] = r.PostForm.Get("username")
			session.Save(r, w)
			w.Header().Set("Location", "/")
			w.WriteHeader(303)
			fmt.Fprint(w, renderPage("You have signed in", ""))

		}
	} else {
		fmt.Fprint(w, showSignIn(""))
	}

}

func SignOutHandler(w http.ResponseWriter, r *http.Request) {
	session := data.GetSession(r)
	if isAuthenticated(*session) {
		session.Values["authenticated"] = false
		session.Save(r, w)
	}
	w.Header().Set("Location", "/signin")
	w.WriteHeader(303)
}

func SignUpHandler(w http.ResponseWriter, r *http.Request) {
	session := data.GetSession(r)
	if isAuthenticated(*session) {
		w.Header().Set("Location", "/")
		w.WriteHeader(303)
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			fmt.Fprint(w, showSignUp(err.Error()))
		} else if err := signUp(r.PostForm); err != nil {
			fmt.Fprint(w, showSignUp(err.Error()))
		} else {
			fmt.Fprint(w, renderPage("Account created", ""))
		}
	} else {
		fmt.Fprint(w, showSignUp(""))
	}
}

func StyleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	fmt.Fprint(w, showStyle())
}
