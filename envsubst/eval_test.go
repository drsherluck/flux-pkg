/*
Copyright 2024 The Flux authors

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

// Forked from: https://github.com/drone/envsubst
// MIT License
// Copyright (c) 2017 drone.io.

package envsubst

import (
	"errors"
	"testing"
)

// test cases sourced from tldp.org
// http://www.tldp.org/LDP/abs/html/parameter-substitution.html

func TestExpand(t *testing.T) {
	var expressions = []struct {
		params map[string]string
		input  string
		output string
	}{
		// text-only
		{
			params: map[string]string{},
			input:  "abcdEFGH28ij",
			output: "abcdEFGH28ij",
		},
		// length
		{
			params: map[string]string{"var01": "abcdEFGH28ij"},
			input:  "${#var01}",
			output: "12",
		},
		// uppercase first
		{
			params: map[string]string{"var01": "abcdEFGH28ij"},
			input:  "${var01^}",
			output: "AbcdEFGH28ij",
		},
		// uppercase
		{
			params: map[string]string{"var01": "abcdEFGH28ij"},
			input:  "${var01^^}",
			output: "ABCDEFGH28IJ",
		},
		// lowercase first
		{
			params: map[string]string{"var01": "ABCDEFGH28IJ"},
			input:  "${var01,}",
			output: "aBCDEFGH28IJ",
		},
		// lowercase
		{
			params: map[string]string{"var01": "ABCDEFGH28IJ"},
			input:  "${var01,,}",
			output: "abcdefgh28ij",
		},
		// substring with position
		{
			params: map[string]string{"path_name": "/home/bozo/ideas/thoughts.for.today"},
			input:  "${path_name:11}",
			output: "ideas/thoughts.for.today",
		},
		// substring with position and length
		{
			params: map[string]string{"path_name": "/home/bozo/ideas/thoughts.for.today"},
			input:  "${path_name:11:5}",
			output: "ideas",
		},
		// default not used
		{
			params: map[string]string{"var": "abc"},
			input:  "${var=xyz}",
			output: "abc",
		},
		// default used
		{
			params: map[string]string{},
			input:  "${var=xyz}",
			output: "xyz",
		},
		{
			params: map[string]string{"default_var": "foo"},
			input:  "something ${var=${default_var}}",
			output: "something foo",
		},
		{
			params: map[string]string{"default_var": "foo1"},
			input:  `foo: ${var=${default_var}-suffix}`,
			output: "foo: foo1-suffix",
		},
		{
			params: map[string]string{"default_var": "foo1"},
			input:  `foo: ${var=prefix${default_var}-suffix}`,
			output: "foo: prefixfoo1-suffix",
		},
		{
			params: map[string]string{},
			input:  "${var:=xyz}",
			output: "xyz",
		},
		// replace suffix
		{
			params: map[string]string{"stringZ": "abcABC123ABCabc"},
			input:  "${stringZ/%abc/XYZ}",
			output: "abcABC123ABCXYZ",
		},
		// replace prefix
		{
			params: map[string]string{"stringZ": "abcABC123ABCabc"},
			input:  "${stringZ/#abc/XYZ}",
			output: "XYZABC123ABCabc",
		},
		// replace all
		{
			params: map[string]string{"stringZ": "abcABC123ABCabc"},
			input:  "${stringZ//abc/xyz}",
			output: "xyzABC123ABCxyz",
		},
		// replace first
		{
			params: map[string]string{"stringZ": "abcABC123ABCabc"},
			input:  "${stringZ/abc/xyz}",
			output: "xyzABC123ABCabc",
		},
		// delete shortest match prefix
		{
			params: map[string]string{"filename": "bash.string.txt"},
			input:  "${filename#*.}",
			output: "string.txt",
		},
		{
			params: map[string]string{"filename": "path/to/file"},
			input:  "${filename#*/}",
			output: "to/file",
		},
		{
			params: map[string]string{"filename": "/path/to/file"},
			input:  "${filename#*/}",
			output: "path/to/file",
		},
		// delete longest match prefix
		{
			params: map[string]string{"filename": "bash.string.txt"},
			input:  "${filename##*.}",
			output: "txt",
		},
		{
			params: map[string]string{"filename": "path/to/file"},
			input:  "${filename##*/}",
			output: "file",
		},
		{
			params: map[string]string{"filename": "/path/to/file"},
			input:  "${filename##*/}",
			output: "file",
		},
		// delete shortest match suffix
		{
			params: map[string]string{"filename": "bash.string.txt"},
			input:  "${filename%.*}",
			output: "bash.string",
		},
		// delete longest match suffix
		{
			params: map[string]string{"filename": "bash.string.txt"},
			input:  "${filename%%.*}",
			output: "bash",
		},

		// nested parameters
		{
			params: map[string]string{"var01": "abcdEFGH28ij"},
			input:  "${var=${var01^^}}",
			output: "ABCDEFGH28IJ",
		},
		// escaped
		{
			params: map[string]string{"var01": "abcdEFGH28ij"},
			input:  "$${var01}",
			output: "${var01}",
		},
		{
			params: map[string]string{"var01": "abcdEFGH28ij"},
			input:  "some text ${var01}$${var$${var01}$var01${var01}",
			output: "some text abcdEFGH28ij${var${var01}$var01abcdEFGH28ij",
		},
		{
			params: map[string]string{"default_var": "foo"},
			input:  "something $${var=${default_var}}",
			output: "something ${var=foo}",
		},
		// some common escaping use cases
		{
			params: map[string]string{"stringZ": "foo/bar"},
			input:  `${stringZ/\//-}`,
			output: "foo-bar",
		},
		{
			params: map[string]string{"stringZ": "foo/bar/baz"},
			input:  `${stringZ//\//-}`,
			output: "foo-bar-baz",
		},
		// escape outside of expansion shouldn't be processed
		{
			params: map[string]string{"default_var": "foo"},
			input:  "\\\\something ${var=${default_var}}",
			output: "\\\\something foo",
		},
		// substitute with a blank string
		{
			params: map[string]string{"stringZ": "foo.bar"},
			input:  `${stringZ/./}`,
			output: "foobar",
		},
	}

	for _, expr := range expressions {
		t.Run(expr.input, func(t *testing.T) {
			t.Log(expr.input)
			output, err := Eval(expr.input, func(s string) (string, bool) {
				return expr.params[s], true
			})
			if err != nil {
				t.Errorf("Want %q expanded but got error %q", expr.input, err)
			}

			if output != expr.output {
				t.Errorf("Want %q expanded to %q, got %q",
					expr.input,
					expr.output,
					output)
			}
		})
	}
}

func TestExpandStrict(t *testing.T) {
	var expressions = []struct {
		params  map[string]string
		input   string
		output  string
		wantErr error
	}{
		// text-only
		{
			params:  map[string]string{},
			input:   "abcdEFGH28ij",
			output:  "abcdEFGH28ij",
			wantErr: nil,
		},
		// existing
		{
			params:  map[string]string{"foo": "bar"},
			input:   "${foo}",
			output:  "bar",
			wantErr: nil,
		},
		// existing string empty
		{
			params:  map[string]string{"foo": ""},
			input:   "${foo}",
			output:  "",
			wantErr: nil,
		},
		// missing
		{
			params:  map[string]string{},
			input:   "${missing}",
			output:  "",
			wantErr: errVarNotSet,
		},
		// missing but has default
		{
			params:  map[string]string{"foo": "bar"},
			input:   "${missing:=default}",
			output:  "default",
			wantErr: nil,
		},
	}

	for _, expr := range expressions {
		t.Run(expr.input, func(t *testing.T) {
			t.Log(expr.input)
			output, err := Eval(expr.input, func(s string) (string, bool) {
				v, exists := expr.params[s]
				return v, exists
			})
			if expr.wantErr == nil && err != nil {
				t.Errorf("Want %q expanded but got error %q", expr.input, err)
			}
			if expr.wantErr != nil && !errors.Is(err, expr.wantErr) {
				t.Errorf("Want error %q but got error %q", expr.wantErr, err)
			}
			if output != expr.output {
				t.Errorf("Want %q expanded to %q, got %q",
					expr.input,
					expr.output,
					output)
			}
		})
	}
}
