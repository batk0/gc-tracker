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

func showStyle() string {
	return `
	.error {
		color: red;
	}
	span {
		padding: 10px;
	}
	.left-margin,.right-margin {
		display: inline-block;
		min-width: 0;
		width: 20%;
	}
	.content {
		display: inline-block;
		width: 1000px;
	}
	h1 {
		text-align: center;
	}
	td {
		background-color: lightgray;
		border-width: 0px 3px;
		border-style: none solid;
		border-color: white;
		padding: 0px 5px;
	}
	td.check {
		background-color: white;
	}
	`
}
