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
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
)

type args struct {
	method string
	uri    string
	form   url.Values
	auth   bool
	reset  bool
}

type want struct {
	code     int
	headers  http.Header
	body     string
	err      error
	cases    map[string]string
	auth     bool
	users    []string
	token    string
	password string
}

type testMatrix []struct {
	name string
	args args
	want want
}

type MockGCTrackerService struct {
	pageError     error
	session       *sessions.Session
	authenticated bool
	casesList     map[string]string
	usersList     map[string]bool
	resetToken    string
	password      string
}

func (*MockGCTrackerService) ShowStyle() string               { return "showStyle" }
func (*MockGCTrackerService) ShowCases() string               { return "showCases" }
func (*MockGCTrackerService) ShowUsers() string               { return "showUsers" }
func (*MockGCTrackerService) ShowSignUp(err string) string    { return "showSignUp" + err }
func (*MockGCTrackerService) ShowSignIn(err string) string    { return "showSignIn" + err }
func (*MockGCTrackerService) ShowResetPwd(err string) string  { return "showResetPwd" + err }
func (*MockGCTrackerService) ShowChangePwd(err string) string { return "showChangePwd" + err }
func (m *MockGCTrackerService) GetResetToken() string         { return m.resetToken }
func (m *MockGCTrackerService) SetResetToken(token string)    { m.resetToken = token }
func (m *MockGCTrackerService) UpdateCases() error            { return m.pageError }
func (m *MockGCTrackerService) IsAuthenticated() bool         { return m.authenticated }
func (m *MockGCTrackerService) SetAuthenticated(auth bool)    { m.authenticated = auth }

func (m *MockGCTrackerService) RenderPage(content, errorMsg string) string {
	return "renderPage " + content + errorMsg
}

func (m *MockGCTrackerService) ResetPwd(r *http.Request) error {
	form := r.PostForm
	username := form.Get("username")
	if username == "" {
		return errors.New("username is empty")
	}
	return nil
}

func (m *MockGCTrackerService) ChangePwd(r *http.Request) error {
	form := r.PostForm
	password := form.Get("password")
	if password == "" {
		return errors.New("password is empty")
	}
	m.password = password
	return nil
}

func (m *MockGCTrackerService) SignIn(form url.Values) error {
	username := form.Get("username")
	if username == "" {
		return errors.New("username is empty")
	} else if !m.usersList[username] {
		return errors.New("user does not exist")
	}
	m.SetAuthenticated(true)
	return nil
}

func (m *MockGCTrackerService) SignUp(form url.Values) error {
	username := form.Get("username")
	if username == "" {
		return errors.New("username is empty")
	} else if m.usersList[username] {
		return errors.New("user already exists")
	}
	m.usersList[username] = true
	return nil
}
func (m *MockGCTrackerService) AddCase(postForm url.Values) {
	m.casesList[postForm["case"][0]] = postForm["name"][0]
}

func (m *MockGCTrackerService) DelCases(cases []string) {
	for _, c := range cases {
		delete(m.casesList, c)
	}
}

func (m *MockGCTrackerService) GetSession(r *http.Request) {
	m.session = sessions.NewSession(sessions.NewCookieStore(), "TESTSESSION")
}

// Helper functions
func assertStatus(t *testing.T, want, got int) {
	t.Helper()
	if want != got {
		t.Errorf("Response code is wrong got %d want %d", got, want)
	}
}

func assertHeaders(t *testing.T, want, got http.Header) {
	t.Helper()
	for k := range want {
		w := want.Get(k)
		if g := got.Get(k); g != w {
			t.Errorf("Header %q is wrong got %q, want %q", k, g, w)
		}
	}
}

func assertBody(t *testing.T, want, got string) {
	t.Helper()
	if want != got {
		t.Errorf("Response body is wrong got %q want %q", got, want)
	}
}

func assertCases(t *testing.T, want map[string]string, got map[string]string) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Response cases list is wrong got %q want %q", got, want)
	}
}

func assertAuthenticated(t *testing.T, want, got bool) {
	t.Helper()
	if want != got {
		t.Errorf("Authenticated got: %t, want: %t", got, want)
	}
}

func assertUsers(t *testing.T, want []string, got map[string]bool) {
	t.Helper()
	for _, w := range want {
		if !got[w] {
			t.Errorf("User %q does not exist", w)
		}
	}
}

func assertPassword(t *testing.T, want, got string) {
	t.Helper()
	if want != got {
		t.Errorf("Wrong password, got: %q want %q", got, want)
	}
}

func assertToken(t *testing.T, want, got string) {
	t.Helper()
	if want != got {
		t.Errorf("Wrong reset token, got: %q want %q", got, want)
	}
}

func TestGCTrackerServer_StyleHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Get valid style",
			args: args{method: http.MethodGet, uri: "/style.css"},
			want: want{
				code: http.StatusOK,
				headers: http.Header{
					"Content-Type": []string{"text/css"},
				},
				body: "showStyle",
			},
		},
		{
			name: "Invalid method",
			args: args{method: http.MethodPost, uri: "/style.css"},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, _ := http.NewRequest(tt.args.method, tt.args.uri, nil)
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{
				pageError: tt.want.err,
			}
			server := NewGCTrackerServer(service)
			server.StyleHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
		})
	}
}

func TestGCTrackerServer_UpdateHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Successful update",
			args: args{method: http.MethodGet, uri: "/update"},
			want: want{
				code: http.StatusOK,
				headers: http.Header{
					"Content-Type": []string{"text/plain; charset=utf-8"},
				},
				body: "OK",
				err:  nil,
			},
		},
		{
			name: "Failed update",
			args: args{method: http.MethodGet, uri: "/update"},
			want: want{
				code: http.StatusBadRequest,
				body: "FAIL",
				err:  errors.New("some error"),
			},
		},
		{
			name: "Invalid method",
			args: args{method: http.MethodPost, uri: "/update"},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, _ := http.NewRequest(tt.args.method, tt.args.uri, nil)
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{
				pageError: tt.want.err,
			}
			server := NewGCTrackerServer(service)
			server.UpdateHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
		})
	}
}

func TestGCTrackerServer_IndexHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Unauthenticated - redirect to /signin",
			args: args{method: http.MethodGet, uri: "/"},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/signin"},
				},
				body: "renderPage Please SignIn first",
			},
		},
		{
			name: "Authenticated - show cases",
			args: args{method: http.MethodGet, uri: "/", auth: true},
			want: want{
				code: http.StatusOK,
				body: "showCases",
			},
		},
		{
			name: "Invalid method",
			args: args{method: http.MethodPost, uri: "/", auth: true},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, _ := http.NewRequest(tt.args.method, tt.args.uri, nil)
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{
				pageError: tt.want.err,
			}
			if tt.args.auth {
				service.SetAuthenticated(true)
			}
			server := NewGCTrackerServer(service)
			server.IndexHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
		})
	}
}

func TestGCTrackerServer_UsersHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Unauthenticated - redirect to /signin",
			args: args{method: http.MethodGet, uri: "/users"},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/signin"},
				},
			},
		},
		{
			name: "Authenticated - show users",
			args: args{method: http.MethodGet, uri: "/users", auth: true},
			want: want{
				code: http.StatusOK,
				body: "showUsers",
			},
		},
		{
			name: "Invalid method",
			args: args{method: http.MethodPost, uri: "/users", auth: true},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, _ := http.NewRequest(tt.args.method, tt.args.uri, nil)
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{
				pageError: tt.want.err,
			}
			if tt.args.auth {
				service.SetAuthenticated(true)
			}
			server := NewGCTrackerServer(service)
			server.UsersHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
		})
	}
}

func TestGCTrackerServer_CaseHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Unauthenticated - redirect to /",
			args: args{method: http.MethodPost, uri: "/case"},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
			},
		},
		{
			name: "Authenticated GET- redirect to /",
			args: args{method: http.MethodGet, uri: "/case", auth: true},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
			},
		},
		{
			name: "Authenticated POST - addCase",
			args: args{
				method: http.MethodPost,
				uri:    "/case",
				auth:   true,
				form: url.Values{
					"add":  []string{"Add"},
					"case": []string{"ABC", "DEF"},
					"name": []string{"abc", "def"},
				},
			},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				cases: map[string]string{
					"INIT": "init",
					"ABC":  "abc",
				},
			},
		},
		{
			name: "Authenticated POST - updateCase",
			args: args{
				method: http.MethodPost,
				uri:    "/case",
				auth:   true,
				form: url.Values{
					"add":  []string{"Add"},
					"case": []string{"INIT"},
					"name": []string{"init1"},
				},
			},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				cases: map[string]string{
					"INIT": "init1",
				},
			},
		},
		{
			name: "Authenticated POST - delCase non-existent",
			args: args{
				method: http.MethodPost,
				uri:    "/case",
				auth:   true,
				form: url.Values{
					"delete": []string{"Delete"},
					"cases":  []string{"ABC", "DEF"},
				},
			},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				cases: map[string]string{
					"INIT": "init",
				},
			},
		},
		{
			name: "Authenticated POST - delCase existent",
			args: args{
				method: http.MethodPost,
				uri:    "/case",
				auth:   true,
				form: url.Values{
					"delete": []string{"Delete"},
					"cases":  []string{"ABC", "INIT", "DEF"},
				},
			},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				cases: map[string]string{},
			},
		},
		{
			name: "Invalid method",
			args: args{method: http.MethodPut, uri: "/case", auth: true},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := strings.NewReader(tt.args.form.Encode())
			request, _ := http.NewRequest(tt.args.method, tt.args.uri, form)
			if tt.args.form != nil {
				request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				request.Header.Add("Content-Length", strconv.Itoa(len(tt.args.form.Encode())))
			}
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{
				pageError: tt.want.err,
				casesList: map[string]string{"INIT": "init"},
			}
			if tt.args.auth {
				service.SetAuthenticated(true)
			}
			server := NewGCTrackerServer(service)
			server.CaseHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
			if tt.want.cases != nil {
				assertCases(t, tt.want.cases, service.casesList)
			}
		})
	}
}

func TestGCTrackerServer_SignOutHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Unauthenticated - redirect to /signin",
			args: args{method: http.MethodGet, uri: "/signout"},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/signin"},
				},
				auth: false,
			},
		},
		{
			name: "Authenticated - unauthorize - redirect to /signin",
			args: args{method: http.MethodGet, uri: "/signout", auth: true},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/signin"},
				},
				auth: false,
			},
		},
		{
			name: "Invalid method - authenticated",
			args: args{method: http.MethodPost, uri: "/signout", auth: true},
			want: want{
				code: http.StatusMethodNotAllowed,
				auth: true,
			},
		},
		{
			name: "Invalid method - unauthenticated",
			args: args{method: http.MethodPost, uri: "/signout"},
			want: want{
				code: http.StatusMethodNotAllowed,
				auth: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, _ := http.NewRequest(tt.args.method, tt.args.uri, nil)
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{
				pageError: tt.want.err,
			}
			if tt.args.auth {
				service.SetAuthenticated(true)
			}
			server := NewGCTrackerServer(service)
			server.SignOutHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
			assertAuthenticated(t, tt.want.auth, service.IsAuthenticated())
		})
	}
}

func TestGCTrackerServer_SignUpHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Unauthenticated - showSignUp",
			args: args{method: http.MethodGet, uri: "/signup"},
			want: want{
				code: http.StatusOK,
				body: "showSignUp",
				auth: false,
			},
		},
		{
			name: "Authenticated - redirect to /",
			args: args{method: http.MethodGet, uri: "/signup", auth: true},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				auth: true,
			},
		},
		{
			name: "Authenticated POST - redirect to /",
			args: args{method: http.MethodPost, uri: "/signup", auth: true},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				auth: true,
			},
		},
		{
			name: "Invalid method",
			args: args{method: http.MethodPut, uri: "/signup", auth: true},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name: "Unauthenticated POST - empty form - showSignUp with error",
			args: args{
				method: http.MethodPost,
				uri:    "/signup",
				auth:   false,
				form:   nil,
			},
			want: want{
				code: http.StatusOK,
				auth: false,
				body: "showSignUpusername is empty",
			},
		},
		{
			name: "Unauthenticated POST - existing user - showSignUp with error",
			args: args{
				method: http.MethodPost,
				uri:    "/signup",
				auth:   false,
				form: url.Values{
					"username": []string{"existing"},
				},
			},
			want: want{
				code: http.StatusOK,
				auth: false,
				body: "showSignUpuser already exists",
			},
		},
		{
			name: "Unauthenticated POST - new user - Account created",
			args: args{
				method: http.MethodPost,
				uri:    "/signup",
				auth:   false,
				form: url.Values{
					"username": []string{"newuser"},
				},
			},
			want: want{
				code:  http.StatusOK,
				auth:  false,
				body:  "renderPage Account created",
				users: []string{"newuser", "existing"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := strings.NewReader(tt.args.form.Encode())
			request, _ := http.NewRequest(tt.args.method, tt.args.uri, form)
			if tt.args.form != nil {
				request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				request.Header.Add("Content-Length", strconv.Itoa(len(tt.args.form.Encode())))
			}
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{
				pageError: tt.want.err,
				usersList: map[string]bool{
					"existing": true,
				},
			}
			if tt.args.auth {
				service.SetAuthenticated(true)
			}
			server := NewGCTrackerServer(service)
			server.SignUpHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
			assertUsers(t, tt.want.users, service.usersList)
		})
	}
}

func TestGCTrackerServer_SignInHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Unauthenticated - showSignIn",
			args: args{method: http.MethodGet, uri: "/signin"},
			want: want{
				code: http.StatusOK,
				body: "showSignIn",
				auth: false,
			},
		},
		{
			name: "Authenticated - redirect to /",
			args: args{method: http.MethodGet, uri: "/signin", auth: true},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				auth: true,
			},
		},
		{
			name: "Authenticated POST - redirect to /",
			args: args{method: http.MethodPost, uri: "/signin", auth: true},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				auth: true,
			},
		},
		{
			name: "Invalid method",
			args: args{method: http.MethodPut, uri: "/signin"},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name: "Unauthenticated POST - empty form - showSignIn with error",
			args: args{
				method: http.MethodPost,
				uri:    "/signin",
				auth:   false,
				form:   nil,
			},
			want: want{
				code: http.StatusOK,
				auth: false,
				body: "showSignInusername is empty",
			},
		},
		{
			name: "Unauthenticated POST - non-existing user - showSignIn with error",
			args: args{
				method: http.MethodPost,
				uri:    "/signin",
				auth:   false,
				form: url.Values{
					"username": []string{"nonexisting"},
				},
			},
			want: want{
				code: http.StatusOK,
				auth: false,
				body: "showSignInuser does not exist",
			},
		},
		{
			name: "Unauthenticated POST - existing user - authorize redirect to /",
			args: args{
				method: http.MethodPost,
				uri:    "/signin",
				auth:   false,
				form: url.Values{
					"username": []string{"existing"},
				},
			},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				auth: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := strings.NewReader(tt.args.form.Encode())
			request, _ := http.NewRequest(tt.args.method, tt.args.uri, form)
			if tt.args.form != nil {
				request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				request.Header.Add("Content-Length", strconv.Itoa(len(tt.args.form.Encode())))
			}
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{
				pageError: tt.want.err,
				usersList: map[string]bool{
					"existing": true,
				},
			}
			service.SetAuthenticated(tt.args.auth)
			server := NewGCTrackerServer(service)
			server.SignInHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
			assertAuthenticated(t, tt.want.auth, service.IsAuthenticated())
		})
	}
}

func TestGCTrackerServer_ResetPwdHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Unauthenticated - showResetPwd",
			args: args{method: http.MethodGet, uri: "/resetpwd"},
			want: want{
				code: http.StatusOK,
				body: "showResetPwd",
				auth: false,
			},
		},
		{
			name: "Authenticated - redirect to /",
			args: args{method: http.MethodGet, uri: "/resetpwd", auth: true},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				auth: true,
			},
		},
		{
			name: "Authenticated POST - redirect to /",
			args: args{method: http.MethodPost, uri: "/resetpwd", auth: true},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/"},
				},
				auth: true,
			},
		},
		{
			name: "Invalid method",
			args: args{method: http.MethodPut, uri: "/resetpwd"},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name: "Unauthenticated POST - empty form - showResetPwd with error",
			args: args{
				method: http.MethodPost,
				uri:    "/resetpwd",
				auth:   false,
				form:   nil,
			},
			want: want{
				code: http.StatusOK,
				auth: false,
				body: "showResetPwdusername is empty",
			},
		},
		{
			name: "Unauthenticated POST - existing user",
			args: args{
				method: http.MethodPost,
				uri:    "/resetpwd",
				auth:   false,
				form: url.Values{
					"username": []string{"existing"},
				},
			},
			want: want{
				code: http.StatusOK,
				auth: false,
				body: "renderPage Check your mailbox for the reset link.",
			},
		},
		{
			name: "Unauthenticated POST - non-existing user",
			args: args{
				method: http.MethodPost,
				uri:    "/resetpwd",
				auth:   false,
				form: url.Values{
					"username": []string{"nonexisting"},
				},
			},
			want: want{
				code: http.StatusOK,
				auth: false,
				body: "renderPage Check your mailbox for the reset link.",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := strings.NewReader(tt.args.form.Encode())
			request, _ := http.NewRequest(tt.args.method, tt.args.uri, form)
			if tt.args.form != nil {
				request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				request.Header.Add("Content-Length", strconv.Itoa(len(tt.args.form.Encode())))
			}
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{}
			service.SetAuthenticated(tt.args.auth)
			server := NewGCTrackerServer(service)
			server.ResetPwdHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
			assertAuthenticated(t, tt.want.auth, service.IsAuthenticated())
		})
	}
}

func TestGCTrackerServer_ChangePwdHandler(t *testing.T) {
	tests := testMatrix{
		{
			name: "Invalid method",
			args: args{method: http.MethodPut, uri: "/changepwd"},
			want: want{
				code:     http.StatusMethodNotAllowed,
				password: "oldpass",
			},
		},
		{
			name: "Unauthenticated GET - redirect to /resetpwd",
			args: args{method: http.MethodGet, uri: "/changepwd"},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/resetpwd"},
				},
				password: "oldpass",
			},
		},
		{
			name: "Authenticated GET - showChangePwd",
			args: args{method: http.MethodGet, uri: "/changepwd", auth: true},
			want: want{
				code:     http.StatusOK,
				body:     "showChangePwd",
				password: "oldpass",
			},
		},
		{
			name: "Unauthenticated GET with token - showChangePwd",
			args: args{method: http.MethodGet, uri: "/changepwd?a=r&t=token"},
			want: want{
				code:     http.StatusOK,
				body:     "showChangePwd",
				token:    "token",
				password: "oldpass",
			},
		},
		{
			name: "Unauthenticated POST - empty form - showChangePwd with error",
			args: args{
				method: http.MethodPost,
				uri:    "/changepwd",
				auth:   false,
				form:   nil,
				reset:  true,
			},
			want: want{
				code:     http.StatusOK,
				body:     "showChangePwdpassword is empty",
				token:    "token",
				password: "oldpass",
			},
		},
		{
			name: "Unauthenticated POST - no token - redirect to /resetpwd",
			args: args{
				method: http.MethodPost,
				uri:    "/changepwd",
				auth:   false,
				form: url.Values{
					"password": []string{"newpass"},
				},
			},
			want: want{
				code: http.StatusSeeOther,
				headers: http.Header{
					"Location": []string{"/resetpwd"},
				},
				password: "oldpass",
			},
		},
		{
			name: "Unauthenticated POST - renderPage Password changed",
			args: args{
				method: http.MethodPost,
				uri:    "/changepwd",
				auth:   false,
				reset:  true,
				form: url.Values{
					"password": []string{"newpass"},
				},
			},
			want: want{
				code:     http.StatusOK,
				body:     "renderPage Password changed",
				token:    "token",
				password: "newpass",
			},
		},
		{
			name: "Authenticated POST - empty form - showChangePwd with error",
			args: args{
				method: http.MethodPost,
				uri:    "/changepwd",
				auth:   true,
				form:   nil,
			},
			want: want{
				code:     http.StatusOK,
				body:     "showChangePwdpassword is empty",
				password: "oldpass",
			},
		},
		{
			name: "Authenticated POST - renderPage Password changed",
			args: args{
				method: http.MethodPost,
				uri:    "/changepwd",
				auth:   true,
				form: url.Values{
					"password": []string{"newpass"},
				},
			},
			want: want{
				code:     http.StatusOK,
				body:     "renderPage Password changed",
				password: "newpass",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := strings.NewReader(tt.args.form.Encode())
			request := httptest.NewRequest(tt.args.method, tt.args.uri, form)
			if tt.args.form != nil {
				request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				request.Header.Add("Content-Length", strconv.Itoa(len(tt.args.form.Encode())))
			}
			response := httptest.NewRecorder()
			service := &MockGCTrackerService{password: "oldpass"}
			service.SetAuthenticated(tt.args.auth)
			if tt.args.reset {
				service.SetResetToken("token")
			}
			server := NewGCTrackerServer(service)
			server.ChangePwdHandler(response, request)

			assertStatus(t, tt.want.code, response.Code)
			assertHeaders(t, tt.want.headers, response.Header())
			assertBody(t, tt.want.body, response.Body.String())
			assertToken(t, tt.want.token, service.GetResetToken())
			assertPassword(t, tt.want.password, service.password)
		})
	}
}
