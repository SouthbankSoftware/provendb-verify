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
 * @Date:   2018-08-07T11:01:25+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-07-09T13:14:44+10:00
 */

package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"testing"

	hasher "github.com/SouthbankSoftware/provendb-verify/pkg/crypto/sha256"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/stretchr/testify/assert"
)

func TestHashBasicBSON(t *testing.T) {
	oid, _ := primitive.ObjectIDFromHex("5b67d1d428d3bf06b1f488ec")
	doc := bsonx.Doc{
		{"_id", bsonx.ObjectID(oid)},
		{
			"_dbproof_metadata",
			bsonx.Document(bsonx.Doc{
				{"_id", bsonx.ObjectID(oid)},
				{"minVersion", bsonx.Int64(2)},
			}),
		},
		{"a", bsonx.Double(1.0)},
		{"b", bsonx.String("provenDB")},
	}
	byteArray, _ := doc.MarshalBSON()

	hash := hasher.HashByteArray(byteArray)

	expectedHash, _ := hex.DecodeString("df90eb334ba90f864631750946ebe1219e2497a36555ed06e3a0bbdc9055e019")
	assert.Equal(t, hash, expectedHash)
}

func TestHashComplexBSON(t *testing.T) {
	doc := bsonx.Doc{}
	bson.UnmarshalExtJSON([]byte(`{
		"_id": {
			"$numberDouble": "1.0"
		},
		"_provendb_metadata": {
			"_id": {
				"$numberDouble": "1.0"
			},
			"minVersion": {
				"$numberLong": "2375"
			}
		},
		"bindata": {
			"$binary": {
				"base64": "c9f0f895fb98ab9159f51fd0297e236d",
				"subType": "00"
			}
		},
		"isodate": {
			"$date": {
				"$numberLong": "1536710248283"
			}
		},
		"timestamp": {
			"$timestamp": {
				"t": 1536710248,
				"i": 1
			}
		},
		"oid": {
			"$oid": "5af11d707d7604ddb14508df"
		},
		"nlong": {
			"$numberLong": "9223372036854775807"
		},
		"decimalQuoted": {
			"$numberDecimal": "123.40"
		}
	}`), true, &doc)
	byteArray, _ := doc.MarshalBSON()

	hash := hasher.HashByteArray(byteArray)

	expectedHash, _ := hex.DecodeString("00c127d9d72cc8bc1782e0cc1ada9bbef3bf73423fa718ea9d801ecfc0af39b2")
	assert.Equal(t, hash, expectedHash)
}

func TestHashBsonByteArray(t *testing.T) {
	// []byte => bson.Document => []byte

	bAIn, err := hex.DecodeString("9b000000015f696400000000000000f03f0562696e6461746100180000000073d7f47fcf797dbf7c69bf75e7d7f9d5f774dbdededb7e9d0969736f646174650082de63f0650100001174696d657374616d70000000000000000000076f6964005af11d707d7604ddb14508df126e6c6f6e6700ffffffffffffff7f13646563696d616c51756f7465640034300000000000000000000000003c3000")
	if err != nil {
		t.Fatal(err)
	}

	doc := bsonx.Doc{}
	doc.UnmarshalBSON(bAIn)

	j, err := bson.MarshalExtJSON(doc, true, false)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("bson.Document in extended JSON:\n%s\n", j)

	hashIn := hasher.HashByteArray(bAIn)
	fmt.Printf("Hash of input []byte:\n%x\n", hashIn)

	bAOut, err := doc.MarshalBSON()
	if err != nil {
		t.Fatal(err)
	}

	hashOut := hasher.HashByteArray(bAOut)
	fmt.Printf("Hash of output []byte:\n%x\n", hashOut)

	docFromJSON := bsonx.Doc{}
	err = bson.UnmarshalExtJSON(j, true, &docFromJSON)
	if err != nil {
		t.Fatal(err)
	}
	bAFromJSON, err := docFromJSON.MarshalBSON()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Hash of []byte parsed from the extended JSON:\n%x\n", hasher.HashByteArray(bAFromJSON))
}

func TestCombineTwoSha256(t *testing.T) {
	aHashStr := "7d536ec0a82aaf6d2e3cdc1b6a1c1d7def3dc3e624305ff82cc0fa7e9a21b926"
	bHashStr := "5044a13e1eaa191436b8bfb19df6a229cba8d64e3d67192a7085f62a94ad3f12"
	aHashBA, _ := hex.DecodeString(aHashStr)
	bHashBA, _ := hex.DecodeString(bHashStr)
	abHashBA, _ := hex.DecodeString(aHashStr + bHashStr)
	combinedHash1 := hasher.HashByteArray(aHashBA, bHashBA)
	combinedHash2 := hasher.HashByteArray(abHashBA)

	expectedHash, _ := hex.DecodeString("caac105b13e7b8eb7cf5ee85a19e73ec01137ea3bcd54061fd37e57f1e23009f")
	assert.Equal(t, combinedHash1, expectedHash)
	assert.Equal(t, combinedHash2, expectedHash)
}

func TestHashDocJSON(t *testing.T) {
	ba, err := ioutil.ReadFile("testdata/excel_example.xlsx.doc.json")
	if err != nil {
		t.Fatal(err)
	}

	doc := bsonx.Doc{}

	err = bson.UnmarshalExtJSON(ba, true, &doc)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = hashDocument(doc)
	if err != nil {
		t.Fatal(err)
	}
}
