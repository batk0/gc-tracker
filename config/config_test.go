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
package config

import (
	"os"
	"reflect"
	"testing"
)

func Test_isAppEngineFunc(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "isAppEngine=true",
			args:    args{v: "AppEngine"},
			want:    IsGAE(true),
			wantErr: false,
		},
		{
			name:    "isAppEngine=false",
			args:    args{v: ""},
			want:    IsGAE(false),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isAppEngineFunc(tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("isAppEngineFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("isAppEngineFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    config
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:    "Empty config",
			want:    config{Port: "8080", Cookie: "sessionid", SMTPPort: "587"},
			wantErr: true,
		},
		{
			name: "Full config",
			env: map[string]string{
				"PORT":         "8880",
				"COOKIE_NAME":  "sessionid",
				"PROJECT_NAME": "PRJ",
				"GAE_ENV":      "AppEngine",
				"SMTP_HOST":    "smtp.example.com",
				"SMTP_PORT":    "465",
				"SMTP_USER":    "user",
				"SMTP_PASS":    "pass",
			},
			want: config{
				Port:        "8880",
				Cookie:      "sessionid",
				Project:     "PRJ",
				IsAppEngine: true,
				SMTPHost:    "smtp.example.com",
				SMTPPort:    "465",
				SMTPUser:    "user",
				SMTPPass:    "pass",
			},
			wantErr: false,
		},
		{
			name: "Missed PROJECT_NAME",
			env: map[string]string{
				"PORT":        "8880",
				"COOKIE_NAME": "sessionid",
				"GAE_ENV":     "AppEngine",
				"SMTP_HOST":   "smtp.example.com",
				"SMTP_PORT":   "465",
				"SMTP_USER":   "user",
				"SMTP_PASS":   "pass",
			},
			want: config{
				Port:        "8880",
				Cookie:      "sessionid",
				IsAppEngine: true,
				SMTPHost:    "smtp.example.com",
				SMTPPort:    "465",
				SMTPUser:    "user",
				SMTPPass:    "pass",
			},
			wantErr: true,
		},
		{
			name: "Missed SMTP_HOST",
			env: map[string]string{
				"PORT":         "8880",
				"COOKIE_NAME":  "sessionid",
				"PROJECT_NAME": "PRJ",
				"GAE_ENV":      "AppEngine",
				"SMTP_PORT":    "465",
				"SMTP_USER":    "user",
				"SMTP_PASS":    "pass",
			},
			want: config{
				Port:        "8880",
				Cookie:      "sessionid",
				Project:     "PRJ",
				IsAppEngine: true,
				SMTPPort:    "465",
				SMTPUser:    "user",
				SMTPPass:    "pass",
			},
			wantErr: true,
		},
		{
			name: "Missed SMTP_USER",
			env: map[string]string{
				"PORT":         "8880",
				"COOKIE_NAME":  "sessionid",
				"PROJECT_NAME": "PRJ",
				"GAE_ENV":      "AppEngine",
				"SMTP_HOST":    "smtp.example.com",
				"SMTP_PORT":    "465",
				"SMTP_PASS":    "pass",
			},
			want: config{
				Port:        "8880",
				Cookie:      "sessionid",
				Project:     "PRJ",
				IsAppEngine: true,
				SMTPHost:    "smtp.example.com",
				SMTPPort:    "465",
				SMTPPass:    "pass",
			},
			wantErr: true,
		},
		{
			name: "Missed SMTP_PASS",
			env: map[string]string{
				"PORT":         "8880",
				"COOKIE_NAME":  "sessionid",
				"PROJECT_NAME": "PRJ",
				"GAE_ENV":      "AppEngine",
				"SMTP_HOST":    "smtp.example.com",
				"SMTP_PORT":    "465",
				"SMTP_USER":    "user",
			},
			want: config{
				Port:        "8880",
				Cookie:      "sessionid",
				Project:     "PRJ",
				IsAppEngine: true,
				SMTPHost:    "smtp.example.com",
				SMTPPort:    "465",
				SMTPUser:    "user",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		setEnvs(t, tt.env)
		Config = config{}
		t.Run(tt.name, func(t *testing.T) {
			if err := InitConfig(); (err != nil) != tt.wantErr {
				t.Errorf("InitConfig() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil && !reflect.DeepEqual(tt.want, Config) {
				t.Errorf("InitConfig() got = %v, want %v", Config, tt.want)
			}
		})
		unsetEnvs(t, tt.env)
	}
}

func setEnvs(t *testing.T, env map[string]string) {
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("cannot set env: %v", err.Error())
		}
	}
}

func unsetEnvs(t *testing.T, env map[string]string) {
	for k := range env {
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("cannot unset env: %v", err.Error())
		}
	}
}
