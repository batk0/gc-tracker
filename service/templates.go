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

var header string = `<!DOCTYPE HTML>
	<html>
	<head>
	<title>GC Tracker</title>
	<link href="/style.css" rel="stylesheet">
	</head>
	<body>
	<span class=left-margin></span>
	<span class=content>
	<h1>Welcome to GC Tracker!</h1>
	`

var footer string = `
	</span>
	<span class=right-margin></span>
	</body>
	</html>
	`

var signout string = `<div><span><a href="/signout">Sign out</a></span><span width=100%>&nbsp;</span><span><a href="/changepwd">Change password</a></span></div>`

func (s *GCTrackerService) ShowSignIn(errorMsg string) string {
	return s.RenderPage(`
	<h2>Sign In</h2>
	<form method=post>
	<div>Username <input type=text name=username></div>
	<div>Password <input type=password name=password></div>
	<div>
	<span><input type=submit value="Sign in"></span>
	<span><a href="/signup">Sign Up</a></span>
	<span><a href="/resetpwd">Forgot password?</a></span>
	</div>
	</form>
	`, errorMsg)
}

func (s *GCTrackerService) ShowSignUp(errorMsg string) string {
	return s.RenderPage(`
<h2>Sign Up</h2>
<form method=post>
<div>Username <input type=text name=username></div>
<div>E-Mail <input type=text name=email></div>	
<div>Password <input type=password name=password></div>
<div>Confirm password <input type=password name=password2></div>
<div>
<span><input type=submit value="Sign Up"></span>
<span><a href="/">Sign In</a></span>
<span><a href="/resetpwd">Forgot password?</a></span>
</div>
</form>
`, errorMsg)
}

func (s *GCTrackerService) ShowChangePwd(errorMsg string) string {
	return s.RenderPage(`<h2>Change password</h2>
	<form method=post>
	<div>Password <input type=password name=password></div>
	<div>Confirm password <input type=password name=password2></div>
	<div>
	<span><input type=submit value="Change password"></span>
	</div>
	</form>
	`, errorMsg)
}

func (s *GCTrackerService) ShowResetPwd(errorMsg string) string {
	return s.RenderPage(`<h2>Reset password</h2>
	<form method=post>
	<div>Username <input type=text name=username></div>
	<div>
	<span><input type=submit value="Reset password"></span>
	<span><a href="/signin">Sign In</a></span>
	<span><a href="/signup">Sign Up</a></span>
	</div>
	</form>
	`, errorMsg)
}

func (s *GCTrackerService) ShowCases() string {
	return s.RenderPage(`<h2>Cases</h2>
	<form method=post action="/case">
	<table>
	`+s.RenderCases()+`
	</table>
	<div>
	<span>ID <input type=text name=case></span>
	<span>Description <input type=text name=name></span>
	</div>
	<div>
	<span><input type=submit name=add value="Add"></span>
	<span><input type=submit name=delete value="Delete"></span>
	</div>
	</form>
	`+signout, "")
}
