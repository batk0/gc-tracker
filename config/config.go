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
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/caarlos0/env"
	"gopkg.in/go-playground/validator.v9"
)

type IsGAE bool

type config struct {
	Port        string `env:"PORT" envDefault:"8080" validate:"required,numeric"`
	Cookie      string `env:"COOKIE_NAME" envDefault:"sessionid" validate:"required,alphanum"`
	Project     string `env:"PROJECT_NAME" validate:"required"`
	IsAppEngine IsGAE  `env:"GAE_ENV"`
	SMTPHost    string `env:"SMTP_HOST" validate:"required"`
	SMTPPort    string `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser    string `env:"SMTP_USER" validate:"required"`
	SMTPPass    string `env:"SMTP_PASS" validate:"required"`
}

var Config = config{}
var isAppEngineType = reflect.TypeOf(Config.IsAppEngine)

func isAppEngineFunc(v string) (interface{}, error) {
	return (IsGAE)(v != ""), nil
}

func InitConfig() error {
	env.ParseWithFuncs(&Config, env.CustomParsers{isAppEngineType: isAppEngineFunc})

	v := validator.New()

	if err := v.Struct(Config); err != nil {
		errorMsg := ""
		for _, e := range err.(validator.ValidationErrors) {
			log.Println(e)
			errorMsg += fmt.Sprintln(e)
		}
		return errors.New(errorMsg)
	}
	return nil
}
