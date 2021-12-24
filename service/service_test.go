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
package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/batk0/gc-tracker/config"
	"github.com/batk0/gc-tracker/data"
	"github.com/gorilla/sessions"
)

type sessionValues map[interface{}]interface{}

type MockGCTrackerData struct {
	cases []*MockGCTrackerCase
	user  *MockGCTrackerUser
}

type MockGCTrackerCase struct {
	err    error
	cnt    func()
	id     string
	name   string
	status string
}

type MockGCTrackerUser struct {
	username     string
	password     string
	notification string
	cases        []*MockGCTrackerCase
	c            data.GCTrackerCase
	caseAdded    bool
	delCaseCnt   int
	casesDeleted int
}

func (u *MockGCTrackerUser) GetUsername() string         { return u.username }
func (u *MockGCTrackerUser) SendNotification(msg string) { u.notification = msg }
func (u *MockGCTrackerUser) SetPassword2(p1, p2 string)  { u.password = p1 }

func (u *MockGCTrackerUser) Authenticate() error {
	if u.username != "existing" {
		return errors.New("user does not exist")
	} else if u.password != "testpassword" {
		return errors.New("password does not match")
	}
	return nil
}

func (u *MockGCTrackerUser) Set(form url.Values) {
	u.username = form.Get("username")
	u.password = form.Get("password")
}

func (u *MockGCTrackerUser) HashAndSalt() error {
	if u.password == "goodpassword" {
		u.password = "hashed"
		return nil
	}
	return errors.New("cannot hash password")
}

func (u *MockGCTrackerUser) Validate(online bool) error {
	if online && u.username == "existing" {
		return errors.New("username already exists")
	}
	if u.username != "gooduser" {
		return errors.New("bad username")
	}
	if u.password != "goodpassword" {
		return errors.New("bad password")
	}
	return nil
}

func (u *MockGCTrackerUser) GetByUsername(username string) error {
	if username != "existing" && username != "gooduser" {
		return errors.New("user not found")
	}
	u.username = username
	if username == "existing" {
		u.cases = []*MockGCTrackerCase{
			{id: "1", name: "case1", status: "status1"},
			{id: "2", name: "case2", status: "status2"},
		}
	}
	return nil
}

func (u *MockGCTrackerUser) GenerateResetToken(addr string) error {
	u.notification = addr + "?a=r&t=token"
	return nil
}

func (u *MockGCTrackerUser) Update() error {
	if u.c != nil {
		u.caseAdded = true
	}
	u.casesDeleted = u.delCaseCnt
	return nil
}

func (u *MockGCTrackerUser) AddCase(c data.GCTrackerCase) error {
	if c.GetID() == "" {
		return errors.New("wrong case id")
	}
	u.c = c
	return nil
}

func (u *MockGCTrackerUser) DelCase(c string) {
	if strings.Contains(c, "existingcase") {
		u.delCaseCnt += 1
	}
}

func (u *MockGCTrackerUser) GetCases() []data.GCTrackerCase {
	cases := []data.GCTrackerCase{}
	for _, c := range u.cases {
		cases = append(cases, c)
	}
	return cases
}

func (d *MockGCTrackerData) NewSession() sessions.Store               { return sessions.NewCookieStore() }
func (d *MockGCTrackerData) CreateUser(user data.GCTrackerUser) error { return nil }
func (d *MockGCTrackerData) UserAvailable(username string) bool       { return username != "existing" }

// Implemented for for compatibility with interface data.GCTrackerData. Not used for tests
func (*MockGCTrackerData) GetCase(string) (*firestore.DocumentSnapshot, error) { return nil, nil }
func (*MockGCTrackerData) GetUsersByCase(string) []data.GCTrackerUser          { return nil }
func (*MockGCTrackerData) CreateCase(data.GCTrackerCase) error                 { return nil }
func (*MockGCTrackerData) DeleteCase(data.GCTrackerCase) error                 { return nil }
func (*MockGCTrackerData) GetCases([]string) []data.GCTrackerCase              { return nil }
func (*MockGCTrackerData) GetUser(string) (*firestore.DocumentSnapshot, error) { return nil, nil }
func (*MockGCTrackerData) UpdateUser(user data.GCTrackerUser) error            { return nil }

func (d *MockGCTrackerData) NewUser() data.GCTrackerUser {
	d.user = &MockGCTrackerUser{}
	return d.user
}

func (d *MockGCTrackerData) GetUserByResetToken(token string) (data.GCTrackerUser, error) {
	if token != "token" {
		return nil, errors.New("token not found")
	}
	d.user = &MockGCTrackerUser{username: "gooduser", password: "oldpassword"}
	return d.user, nil
}

func (d *MockGCTrackerData) GetAllCases() []data.GCTrackerCase {
	cases := []data.GCTrackerCase{}
	for _, c := range d.cases {
		cases = append(cases, c)
	}
	return cases
}

func (d *MockGCTrackerData) NewCase() data.GCTrackerCase {
	c := &MockGCTrackerCase{}
	return c
}

func (c *MockGCTrackerCase) Create()            { c.cnt() }
func (c *MockGCTrackerCase) CheckStatus() error { return c.err }
func (c *MockGCTrackerCase) GetID() string      { return c.id }
func (c *MockGCTrackerCase) GetName() string    { return c.name }
func (c *MockGCTrackerCase) GetStatus() string  { return c.status }
func (c *MockGCTrackerCase) Validate() error    { return nil }

func (c *MockGCTrackerCase) Set(f url.Values) {
	c.id = f.Get("case")
	c.name = f.Get("name")
}

/* Helper functions */
func postForm(uri string, values url.Values) *http.Request {
	request := httptest.NewRequest(http.MethodPost, uri, nil)
	request.PostForm = values
	return request
}

func assertSession(t *testing.T, got *sessions.Session, want *sessions.Session) {
	t.Helper()
	err := false
	if want == nil {
		err = got != want
	} else {
		err = got == nil || got.Name() != want.Name()
	}
	if err {
		t.Errorf("GCTrackerService.session = %v, want %v", got, want)
	}
}

func setSession(v map[interface{}]interface{}) *sessions.Session {
	s := sessions.NewSession(sessions.NewCookieStore(), "TESTSESSION")
	s.Values = v
	return s
}

func TestGCTrackerService_RenderPage(t *testing.T) {
	type args struct {
		content  string
		errorMsg string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Render no error",
			args: args{content: "page", errorMsg: ""},
			want: "header page footer",
		},
		{
			name: "Render with error",
			args: args{content: "page", errorMsg: "error"},
			want: "header <div class='error'><ul><li>error</ul></div>page footer",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &GCTrackerService{}
			header = "header "
			footer = " footer"
			if got := s.RenderPage(tt.args.content, tt.args.errorMsg); got != tt.want {
				t.Errorf("GCTrackerService.RenderPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGCTrackerService_GetSession(t *testing.T) {

	type args struct {
		r      *http.Request
		cookie string
	}
	tests := []struct {
		name string
		args args
		want *sessions.Session
	}{
		{
			name: "GetSession empty cookie",
			args: args{r: httptest.NewRequest(http.MethodGet, "/", nil), cookie: ""},
			want: nil,
		},
		{
			name: "GetSession non-empty cookie",
			args: args{r: httptest.NewRequest(http.MethodGet, "/", nil), cookie: "TESTSESSION"},
			want: sessions.NewSession(sessions.NewCookieStore(), "TESTSESSION"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewGCTrackerService(&MockGCTrackerData{})
			config.Config.Cookie = tt.args.cookie
			s.GetSession(tt.args.r)

			assertSession(t, s.session, tt.want)
		})
	}
}

func TestGCTrackerService_IsAuthenticated(t *testing.T) {
	tests := []struct {
		name    string
		session *sessions.Session
		want    bool
	}{
		{
			name:    "Empty session",
			session: nil,
			want:    false,
		},
		{
			name:    "Empty session values",
			session: setSession(nil),
			want:    false,
		},
		{
			name:    "Unauthenticated with username",
			session: setSession(sessionValues{"username": "user"}),
			want:    false,
		},
		{
			name:    "Authenticated without username",
			session: setSession(sessionValues{"authenticated": true}),
			want:    false,
		},
		{
			name: "Authenticated with username",
			session: setSession(sessionValues{
				"username":      "user",
				"authenticated": true,
			}),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &GCTrackerService{
				session: tt.session,
			}
			if got := s.IsAuthenticated(); got != tt.want {
				t.Errorf("GCTrackerService.IsAuthenticated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGCTrackerService_UpdateCases(t *testing.T) {
	createCnt := 0
	cntFunc := func() {
		createCnt += 1
	}
	tests := []struct {
		name      string
		cases     []*MockGCTrackerCase
		wantErr   bool
		createCnt int
	}{
		{
			name:      "No cases",
			wantErr:   false,
			createCnt: 0,
		},
		{
			name:      "Update successfull",
			cases:     []*MockGCTrackerCase{{cnt: cntFunc}, {cnt: cntFunc}},
			wantErr:   false,
			createCnt: 0,
		},
		{
			name:      "Status Check = status changed",
			cases:     []*MockGCTrackerCase{{err: errors.New("status changed"), cnt: cntFunc}, {cnt: cntFunc}},
			wantErr:   false,
			createCnt: 1,
		},
		{
			name:      "Status Check failed",
			cases:     []*MockGCTrackerCase{{err: errors.New("fail"), cnt: cntFunc}, {cnt: cntFunc}},
			wantErr:   true,
			createCnt: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createCnt = 0
			s := &GCTrackerService{
				data: &MockGCTrackerData{
					cases: tt.cases,
				},
			}
			if err := s.UpdateCases(); (err != nil) != tt.wantErr {
				t.Errorf("GCTrackerService.UpdateCases() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.createCnt != createCnt {
				t.Errorf("GCTrackerService.UpdateCases() creted = %d, want %d", createCnt, tt.createCnt)
			}
		})
	}
}

func TestGCTrackerService_SignIn(t *testing.T) {
	tests := []struct {
		name    string
		args    url.Values
		wantErr bool
	}{
		{
			name:    "Empty form",
			args:    nil,
			wantErr: true,
		},
		{
			name: "Wrong user",
			args: url.Values{
				"username": []string{"nonexisting"},
				"password": []string{"testpassword"},
			},
			wantErr: true,
		},
		{
			name: "Wrong password",
			args: url.Values{
				"username": []string{"existing"},
				"password": []string{"wrongpassword"},
			},
			wantErr: true,
		},
		{
			name: "Successful sign in",
			args: url.Values{
				"username": []string{"existing"},
				"password": []string{"testpassword"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &GCTrackerService{
				data: &MockGCTrackerData{},
			}
			if err := s.SignIn(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("GCTrackerService.SignIn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGCTrackerService_SignUp(t *testing.T) {
	tests := []struct {
		name             string
		args             url.Values
		wantErr          bool
		wantNotification string
	}{
		{
			name:    "Empty form",
			args:    nil,
			wantErr: true,
		},
		{
			name: "Bad username",
			args: url.Values{
				"username": []string{"baduser"},
				"password": []string{"goodpassword"},
			},
			wantErr: true,
		},
		{
			name: "Bad password",
			args: url.Values{
				"username": []string{"gooduser"},
				"password": []string{"badpassword"},
			},
			wantErr: true,
		},
		{
			name: "Existing username",
			args: url.Values{
				"username": []string{"existing"},
				"password": []string{"goodpassword"},
			},
			wantErr: true,
		},
		{
			name: "Successful signup",
			args: url.Values{
				"username": []string{"gooduser"},
				"password": []string{"goodpassword"},
			},
			wantErr:          false,
			wantNotification: "Your account 'gooduser' has been created.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &MockGCTrackerData{}
			s := &GCTrackerService{data: d}
			if err := s.SignUp(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("GCTrackerService.SignUp() error = %v, wantErr %v", err, tt.wantErr)
			}
			if d.user.notification != tt.wantNotification {
				t.Errorf("GCTrackerService.SignUp() notification = %v, want %v", d.user.notification, tt.wantNotification)
			}
		})
	}
}

func TestGCTrackerService_ResetPwd(t *testing.T) {
	tests := []struct {
		name             string
		args             *http.Request
		wantErr          bool
		wantNotification string
	}{
		{
			name:    "Empty form",
			args:    postForm("/resetpwd", nil),
			wantErr: true,
		},
		{
			name: "Non existing user",
			args: postForm("/resetpwd", url.Values{
				"username": []string{"nonexisting"},
			}),
			wantErr: true,
		},
		{
			name: "Existing user",
			args: postForm("/resetpwd", url.Values{
				"username": []string{"existing"},
			}),
			wantErr:          false,
			wantNotification: "/changepwd?a=r&t=token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &MockGCTrackerData{}
			s := &GCTrackerService{data: d}
			if err := s.ResetPwd(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("GCTrackerService.ResetPwd() error = %v, wantErr %v", err, tt.wantErr)
			}
			if d.user != nil && d.user.notification != tt.wantNotification {
				t.Errorf("GCTrackerService.ResetPwd() notification = %q, want %q", d.user.notification, tt.wantNotification)
			}
		})
	}
}

func TestGCTrackerService_ChangePwd(t *testing.T) {
	type want struct {
		err          bool
		notification string
		password     string
	}
	type args struct {
		r *http.Request
		s *sessions.Session
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Empty session and form",
			args: args{s: setSession(nil)},
			want: want{err: true},
		},
		{
			name: "User with reset token",
			args: args{
				r: postForm("/changepwd", url.Values{
					"password":  []string{"goodpassword"},
					"password2": []string{"goodpassword"},
				}),
				s: setSession(sessionValues{"resetToken": "token"}),
			},
			want: want{
				err:          false,
				notification: "Your password has been changed.",
				password:     "hashed",
			},
		},
		{
			name: "Authenticated user",
			args: args{
				r: postForm("/changepwd", url.Values{
					"password":  []string{"goodpassword"},
					"password2": []string{"goodpassword"},
				}),
				s: setSession(sessionValues{
					"authenticated": true,
					"username":      "gooduser",
				}),
			},
			want: want{
				err:          false,
				notification: "Your password has been changed.",
				password:     "hashed",
			},
		},
		{
			name: "User with reset token - bad password",
			args: args{
				r: postForm("/changepwd", url.Values{
					"password":  []string{"badpassword"},
					"password2": []string{"badpassword"},
				}),
				s: setSession(sessionValues{"resetToken": "token"}),
			},
			want: want{
				err:      true,
				password: "badpassword",
			},
		},
		{
			name: "Authenticated user - bad password",
			args: args{
				r: postForm("/changepwd", url.Values{
					"password":  []string{"badpassword"},
					"password2": []string{"badpassword"},
				}),
				s: setSession(sessionValues{
					"authenticated": true,
					"username":      "gooduser",
				}),
			},
			want: want{
				err:      true,
				password: "badpassword",
			},
		},
		{
			name: "Unauthenticated user ",
			args: args{
				r: postForm("/changepwd", url.Values{
					"password":  []string{"goodpassword"},
					"password2": []string{"goodpassword"},
				}),
				s: setSession(sessionValues{
					"authenticated": false,
					"username":      "gooduser",
				}),
			},
			want: want{
				err: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &MockGCTrackerData{}
			s := &GCTrackerService{session: tt.args.s, data: d}
			if err := s.ChangePwd(tt.args.r); (err != nil) != tt.want.err {
				t.Errorf("GCTrackerService.ChangePwd() error = %v, wantErr %v", err, tt.want.err)
			}
			if d.user != nil && d.user.notification != tt.want.notification {
				t.Errorf("GCTrackerService.ChangePwd() notification = %q, want %q", d.user.notification, tt.want.notification)
			}
			if d.user != nil && d.user.password != tt.want.password {
				t.Errorf("GCTrackerService.ChangePwd() password = %q, want %q", d.user.password, tt.want.password)
			}
		})
	}
}

func TestGCTrackerService_AddCase(t *testing.T) {
	type args struct {
		formData url.Values
		s        *sessions.Session
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Empty session",
			args: args{s: setSession(nil)},
			want: false,
		},
		{
			name: "Empty form",
			args: args{s: setSession(sessionValues{"username": "existing"})},
			want: false,
		},
		{
			name: "Non existing user",
			args: args{
				s: setSession(sessionValues{"username": "nonexisting"}),
				formData: url.Values{
					"case": []string{"casenumber"},
					"name": []string{"caseexample"},
				},
			},
			want: false,
		},
		{
			name: "Add Case",
			args: args{
				s: setSession(sessionValues{"username": "existing"}),
				formData: url.Values{
					"case": []string{"casenumber"},
					"name": []string{"caseexample"},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &MockGCTrackerData{}
			s := &GCTrackerService{session: tt.args.s, data: d}
			s.AddCase(tt.args.formData)
			if d.user.caseAdded != tt.want {
				t.Errorf("GCTrackerService.AddCase() caseAdded = %v, want %v", d.user.caseAdded, tt.want)
			}
		})
	}
}

func TestGCTrackerService_DelCases(t *testing.T) {
	type args struct {
		cases []string
		s     *sessions.Session
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Empty session",
			args: args{
				s:     setSession(nil),
				cases: []string{"existingcase"},
			},
			want: 0,
		},
		{
			name: "Empty cases",
			args: args{
				s:     setSession(sessionValues{"username": "existing"}),
				cases: []string{},
			},
			want: 0,
		},
		{
			name: "Delete 1 case",
			args: args{
				s:     setSession(sessionValues{"username": "existing"}),
				cases: []string{"existingcase"},
			},
			want: 1,
		},
		{
			name: "Delete 2 cases",
			args: args{
				s:     setSession(sessionValues{"username": "existing"}),
				cases: []string{"existingcase", "existingcase2"},
			},
			want: 2,
		},
		{
			name: "Delete 2 cases, 1 bad case",
			args: args{
				s:     setSession(sessionValues{"username": "existing"}),
				cases: []string{"bad case", "existingcase", "existingcase2"},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &MockGCTrackerData{}
			s := &GCTrackerService{session: tt.args.s, data: d}
			s.DelCases(tt.args.cases)
			if d.user.casesDeleted != tt.want {
				t.Errorf("GCTrackerService.DelCases() caseAdded = %v, want %v", d.user.casesDeleted, tt.want)
			}
		})
	}
}

func TestGCTrackerService_RenderCases(t *testing.T) {
	type args struct {
		s *sessions.Session
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			name: "Emtpy session",
			args: args{s: setSession(nil)},
			want: "",
		},
		{
			name: "Non-existing user",
			args: args{s: setSession(sessionValues{"username": "nonexisting"})},
			want: "",
		},
		{
			name: "Existing user with cases",
			args: args{s: setSession(sessionValues{"username": "existing"})},
			want: "<tr><td class=check><input type=checkbox name=cases value=1></td><td>1</td><td>case1</td><td>status1</td></tr><tr><td class=check><input type=checkbox name=cases value=2></td><td>2</td><td>case2</td><td>status2</td></tr>",
		},
		{
			name: "Existing user without cases",
			args: args{s: setSession(sessionValues{"username": "gooduser"})},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &MockGCTrackerData{}
			s := &GCTrackerService{session: tt.args.s, data: d}
			if got := s.RenderCases(); got != tt.want {
				t.Errorf("GCTrackerService.RenderCases() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGCTrackerService_RenderError(t *testing.T) {
	type args struct {
		errorMsg string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "No error",
			args: args{""},
			want: "",
		},
		{
			name: "One error",
			args: args{"error"},
			want: "<div class='error'><ul><li>error</ul></div>",
		},
		{
			name: "Two errors",
			args: args{"error1\nerror2"},
			want: "<div class='error'><ul><li>error1<li>error2</ul></div>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &MockGCTrackerData{}
			s := NewGCTrackerService(d)
			if got := s.RenderError(tt.args.errorMsg); got != tt.want {
				t.Errorf("GCTrackerService.RenderError() = %v, want %v", got, tt.want)
			}
		})
	}
}
