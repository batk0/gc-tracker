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
	"net/url"

	"github.com/batk0/gc-tracker/service"
)

type GCTrackerService interface {
	RenderPage(string, string) string
	ShowStyle() string
	ShowCases() string
	ShowUsers() string
	ShowSignIn(string) string
	ShowSignUp(string) string
	ShowResetPwd(string) string
	ShowChangePwd(string) string

	SignIn(url.Values) error
	SignUp(url.Values) error
	ChangePwd(*http.Request) error
	ResetPwd(*http.Request) error
	AddCase(url.Values)
	DelCases([]string)
	UpdateCases() error
	IsAuthenticated() bool
	GetSession(*http.Request)
	SetAuthenticated(bool)
	SetResetToken(string)
	GetResetToken() string
}

type GCTrackerServer struct{ service GCTrackerService }

func NewGCTrackerServer(s GCTrackerService) *GCTrackerServer {
	if s == nil {
		s = service.NewGCTrackerService(nil)
	}
	return &GCTrackerServer{service: s}
}

func (s *GCTrackerServer) CaseHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.service.GetSession(r)
	if s.service.IsAuthenticated() && r.Method == http.MethodPost {
		defer r.Body.Close()
		if err := r.ParseForm(); err != nil {
			log.Println(err.Error())
		} else if r.PostForm.Get("add") != "" {
			s.service.AddCase(r.PostForm)
		} else if r.PostForm.Get("delete") != "" {
			s.service.DelCases(r.PostForm["cases"])
		}
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}

func (s *GCTrackerServer) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.service.GetSession(r)
		if s.service.IsAuthenticated() {
			fmt.Fprint(w, s.service.ShowCases())
		} else {
			w.Header().Set("Location", "/signin")
			w.WriteHeader(http.StatusSeeOther)
			fmt.Fprint(w, s.service.RenderPage("Please SignIn first", ""))
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *GCTrackerServer) UsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.service.GetSession(r)
		if s.service.IsAuthenticated() {
			fmt.Fprint(w, s.service.ShowUsers())
		} else {
			w.Header().Set("Location", "/signin")
			w.WriteHeader(http.StatusSeeOther)
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *GCTrackerServer) ResetPwdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodPost {
		s.service.GetSession(r)
		if s.service.IsAuthenticated() {
			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusSeeOther)
		} else if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				fmt.Fprint(w, s.service.ShowResetPwd(err.Error()))
			} else if err := s.service.ResetPwd(r); err != nil {
				fmt.Fprint(w, s.service.ShowResetPwd(err.Error()))
			} else {
				fmt.Fprint(w, s.service.RenderPage("Check your mailbox for the reset link.", ""))
			}
		} else {
			fmt.Fprint(w, s.service.ShowResetPwd(""))
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *GCTrackerServer) ChangePwdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodPost {
		s.service.GetSession(r)
		if s.service.IsAuthenticated() {
			if r.Method == http.MethodPost {
				if err := r.ParseForm(); err != nil {
					fmt.Fprint(w, s.service.ShowChangePwd(err.Error()))
				} else if err := s.service.ChangePwd(r); err != nil {
					fmt.Fprint(w, s.service.ShowChangePwd(err.Error()))
				} else {
					fmt.Fprint(w, s.service.RenderPage("Password changed", ""))
				}
			} else {
				fmt.Fprint(w, s.service.ShowChangePwd(""))
			}
		} else {
			if r.Method == http.MethodPost && s.service.GetResetToken() != "" {
				if err := r.ParseForm(); err != nil {
					fmt.Fprint(w, s.service.ShowChangePwd(err.Error()))
				} else if err := s.service.ChangePwd(r); err != nil {
					fmt.Fprint(w, s.service.ShowChangePwd(err.Error()))
				} else {
					fmt.Fprint(w, s.service.RenderPage("Password changed", ""))
				}
			} else if q, err := url.ParseQuery(r.RequestURI); err == nil {
				if token := q.Get("t"); token != "" {
					s.service.SetResetToken(token)
					fmt.Fprint(w, s.service.ShowChangePwd(""))
					return
				}
			}
			w.Header().Set("Location", "/resetpwd")
			w.WriteHeader(http.StatusSeeOther)
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *GCTrackerServer) SignInHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodPost {
		s.service.GetSession(r)
		if s.service.IsAuthenticated() {
			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusSeeOther)
		} else if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				fmt.Fprint(w, s.service.ShowSignIn(err.Error()))
			} else if err := s.service.SignIn(r.PostForm); err != nil {
				fmt.Fprint(w, s.service.ShowSignIn(err.Error()))
			} else {
				s.service.SetAuthenticated(true)
				w.Header().Set("Location", "/")
				w.WriteHeader(http.StatusSeeOther)
			}
		} else {
			fmt.Fprint(w, s.service.ShowSignIn(""))
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *GCTrackerServer) SignOutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.service.GetSession(r)
		if s.service.IsAuthenticated() {
			s.service.SetAuthenticated(false)
		}
		w.Header().Set("Location", "/signin")
		w.WriteHeader(http.StatusSeeOther)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *GCTrackerServer) SignUpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodPost {
		s.service.GetSession(r)
		if s.service.IsAuthenticated() {
			w.Header().Set("Location", "/")
			w.WriteHeader(http.StatusSeeOther)
		} else if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				fmt.Fprint(w, s.service.ShowSignUp(err.Error()))
			} else if err := s.service.SignUp(r.PostForm); err != nil {
				fmt.Fprint(w, s.service.ShowSignUp(err.Error()))
			} else {
				fmt.Fprint(w, s.service.RenderPage("Account created", ""))
			}
		} else {
			fmt.Fprint(w, s.service.ShowSignUp(""))
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *GCTrackerServer) StyleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/css")
		fmt.Fprint(w, s.service.ShowStyle())
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *GCTrackerServer) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if err := s.service.UpdateCases(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println(err.Error())
			fmt.Fprint(w, "FAIL")
		} else {
			fmt.Fprint(w, "OK")
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
