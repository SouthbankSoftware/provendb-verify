/*
 * provendb-verify
 * Copyright (C) 2019  Southbank Software Ltd.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 *
 * @Author: guiguan
 * @Date:   2018-08-22T14:32:42+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-04-02T13:31:52+11:00
 */

package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/xeipuuv/gojsonschema"
)

func getTestDataPath(t *testing.T, name string) string {
	path, err := filepath.Abs(filepath.Join("../testdata", name))
	if err != nil {
		t.Fatal(err)
	}

	if _, err = os.Stat(path); os.IsNotExist(err) {
		// try current directory
		path, err = filepath.Abs(filepath.Join("./testdata", name))
		if err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}

	return path
}

// LoadJSON loads given test data into JSON interface{}
func LoadJSON(t *testing.T, name string) interface{} {
	path := getTestDataPath(t, name)

	jsonLoader := gojsonschema.NewReferenceLoader("file://" + path)
	json, err := jsonLoader.LoadJSON()
	if err != nil {
		t.Fatal(err)
	}

	return json
}

// LoadFile loads given test data into *os.File
func LoadFile(t *testing.T, name string) *os.File {
	path := getTestDataPath(t, name)

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	return f
}

// LoadString loads given test data into string
func LoadString(t *testing.T, name string) string {
	path := getTestDataPath(t, name)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}
