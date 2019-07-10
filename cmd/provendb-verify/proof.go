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
 * @Date:   2019-04-02T13:39:00+11:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-07-10T17:12:33+10:00
 */

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/SouthbankSoftware/provendb-verify/pkg/merkle"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/binary"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/eval"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

func getProofType(proof interface{}) (proofType proofType, err error) {
	defer func() {
		if r := recover(); r != nil {
			proofType = ""
			err = r.(error)
		}
	}()

	switch label := proof.(map[string]interface{})["branches"].([]interface{})[0].(map[string]interface{})["label"]; label {
	case provenDBDocBranch:
		proofType = proofTypes.document
	default:
		proofType = proofTypes.database
	}

	return
}

func docProof2DBProof(docProof interface{}) (dbProof interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			docProof = nil
			err = r.(error)
		}
	}()

	proofMap := docProof.(map[string]interface{})
	branches := proofMap["branches"].([]interface{})
	branch := branches[0].(map[string]interface{})
	if branch["label"] != provenDBDocBranch {
		err = fmt.Errorf("the input Chainpoint Proof is not a document Proof")
		return
	}

	startHash, err := hex.DecodeString(proofMap["hash"].(string))
	if err != nil {
		return
	}
	_, endHash := eval.Branch(startHash, branch)
	proofMap["hash"] = hex.EncodeToString(endHash)
	proofMap["branches"] = branches[1:]

	return proofMap, nil
}

func dbProof2DocProof(dbProof interface{}, docMklPrf merkle.Proof) (docProof interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			docProof = nil
			err = r.(error)
		}
	}()

	var hash []byte

	switch algo := docMklPrf.ValueHashAlgorithm; algo {
	case merkle.VHAS.None:
		hash = docMklPrf.Value
	default:
		return nil, fmt.Errorf("%s is not a supported value hash algorithm", algo)
	}

	var hca string

	switch algo := docMklPrf.HashCombiningAlgorithm; algo {
	case merkle.HCAS.Sha256:
		hca = "sha-256"
	default:
		return nil, fmt.Errorf("%s is not a supported hash combining algorithm", algo)
	}

	dbProofMap := dbProof.(map[string]interface{})
	dbProofMap["hash"] = hex.EncodeToString(hash)

	branch := make(map[string]interface{})

	branch["label"] = provenDBDocBranch

	ops := make([]interface{}, 0, len(docMklPrf.Path)*2)

	for _, p := range docMklPrf.Path {
		if len(p.LeftHash) > 0 {
			ops = append(ops, map[string]interface{}{
				"l": hex.EncodeToString(p.LeftHash),
			})
		} else {
			ops = append(ops, map[string]interface{}{
				"r": hex.EncodeToString(p.RightHash),
			})
		}

		ops = append(ops, map[string]interface{}{
			"op": hca,
		})
	}

	branch["ops"] = ops

	branches := dbProofMap["branches"].([]interface{})
	branches = append([]interface{}{branch}, branches...)
	dbProofMap["branches"] = branches

	return dbProof, nil
}

func loadProof(filename string) (proof interface{}, err error) {
	defer func() {
		if err != nil {
			proof = nil
			err = fmt.Errorf("cannot load Chainpoint Proof from `%s`: %s", filename, err)
		}
	}()

	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	if strings.HasSuffix(filename, ".json") {
		var data []byte
		data, err = ioutil.ReadAll(f)
		if err != nil {
			return
		}

		err = json.Unmarshal(data, &proof)
		if err != nil {
			return
		}
	} else {
		proof, err = binary.Base642Proof(f)
		if err != nil {
			return
		}
	}

	return
}

func saveProof(filename string, proof interface{}) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot save Chainpoint Proof to `%s`: %s", filename, err)
		}
	}()

	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()

	if strings.HasSuffix(filename, ".json") {
		var data []byte
		data, err = json.MarshalIndent(proof, "", "  ")
		if err != nil {
			return
		}

		_, err = f.Write(data)
		if err != nil {
			return
		}
	} else {
		err = binary.Proof2Base64(proof, f)
		if err != nil {
			return
		}
	}

	return
}

// getProof gets a Chainpoint Proof and its associated version stored in ProvenDB using either a
// `proofId` (string) or a `versionId` (int64)
func getProof(ctx context.Context, database *mongo.Database, id interface{}, colName string) (
	proof interface{}, version int64, cols []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	doc := bsonx.Doc{}
	var searchEl bsonx.Elem

	switch i := id.(type) {
	case string:
		searchEl = bsonx.Elem{provenDBProofIDKey, bsonx.String(i)}
	case int64:
		searchEl = bsonx.Elem{provenDBVersionKey, bsonx.Int64(i)}
	default:
		err = fmt.Errorf("unsupported ID type %T for getProof", i)
		return
	}

	filter := bsonx.Doc{
		searchEl,
		{provenDBStatusKey, bsonx.Document(bsonx.Doc{
			{"$in", bsonx.Array(bsonx.Arr{
				bsonx.String("submitted"),
				bsonx.String("valid"),
			})},
		})},
	}

	if colName != "" {
		// we have to make sure that the proof covers the given colName
		filter = append(filter, bsonx.Elem{
			provenDBDetailsCollectionsKey,
			bsonx.Document(bsonx.Doc{
				{"$elemMatch", bsonx.Document(bsonx.Doc{
					{"name", bsonx.String(colName)},
				})},
			}),
		})
	}

	err = database.
		Collection(provenDBVersionProofs).
		FindOne(ctx,
			filter,
			options.FindOne().
				SetSort(bsonx.Doc{
					{provenDBStatusKey, bsonx.Int32(-1)},
					{provenDBSubmittedKey, bsonx.Int32(1)},
				}).
				SetProjection(
					bsonx.Doc{
						{provenDBProofIDKey, bsonx.Int32(1)},
						{provenDBVersionKey, bsonx.Int32(1)},
						{provenDBStatusKey, bsonx.Int32(1)},
						{provenDBProofKey, bsonx.Int32(1)},
						{provenDBDetailsKey, bsonx.Int32(1)},
						{provenDBScopeKey, bsonx.Int32(1)},
					},
				),
		).
		Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			colMsg := ""

			if colName != "" {
				colMsg = fmt.Sprintf(" that covers collection `%s`", colName)
			}

			err = fmt.Errorf("no proof%s with status `submitted` or `valid` can be found", colMsg)
		}
		return
	}

	fmt.Printf("Loading Chainpoint Proof `%s`...\n", doc.Lookup(provenDBProofIDKey))

	version, ok := doc.Lookup(provenDBVersionKey).Int64OK()
	if !ok {
		err = fmt.Errorf("cannot get %s", provenDBVersionKey)
		return
	}

	_, proofBytes := doc.Lookup(provenDBProofKey).Binary()

	proof, err = binary.Binary2Proof(bytes.NewBuffer(proofBytes))
	if err != nil {
		return
	}

	scope, ok := doc.Lookup(provenDBScopeKey).StringValueOK()
	if !ok {
		err = fmt.Errorf("cannot get %s", provenDBScopeKey)
		return
	}

	if scope == provenDBScopeCollection {
		arr, ok := doc.Lookup(provenDBDetailsKey, provenDBCollectionsKey).ArrayOK()
		if !ok {
			err = fmt.Errorf("cannot get %s.%s", provenDBDetailsKey, provenDBCollectionsKey)
			return
		}

		for _, val := range arr {
			doc, ok := val.DocumentOK()
			if !ok {
				err = fmt.Errorf("invalid doc in %s.%s", provenDBDetailsKey,
					provenDBCollectionsKey)
				return
			}

			name, ok := doc.Lookup(provenDBNameKey).StringValueOK()
			if !ok {
				err = fmt.Errorf("cannot get %s", provenDBNameKey)
				return
			}

			cols = append(cols, name)
		}
	}

	return
}
