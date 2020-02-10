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
 * @Date:   2019-04-02T13:40:08+11:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-04-02T13:40:57+11:00
 */

package main

import (
	"context"
	"fmt"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"strings"
	"time"

	"github.com/SouthbankSoftware/provendb-verify/pkg/merkle"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

func getLatestVerifiableVersion(ctx context.Context, database *mongo.Database) (int64, error) {
	doc := bsonx.Doc{}

	err := database.
		Collection(provenDBVersionProofs).
		FindOne(ctx, bsonx.Doc{},
			options.FindOne().
				SetSort(bsonx.Doc{
					{provenDBVersionKey, bsonx.Int32(-1)},
					{provenDBSubmittedKey, bsonx.Int32(1)},
				}).
				SetProjection(bsonx.Doc{
					{provenDBVersionKey, bsonx.Int32(1)},
				}),
		).
		Decode(&doc)
	if err != nil {
		return 0, err
	}

	result, ok := doc.Lookup(provenDBVersionKey).Int64OK()
	if !ok {
		return 0, fmt.Errorf("cannot get %s in returned BSON doc", provenDBVersionKey)
	}

	return result, nil
}

type verifiableVersion struct {
	proofID         string
	versionID       int64
	submitTimestamp time.Time
	proofStatus     string
}

func getVerifiableVersions(ctx context.Context, database *mongo.Database) ([]verifiableVersion, error) {
	cur, err := database.
		Collection(provenDBVersionProofs).
		Find(ctx, bsonx.Doc{},
			options.Find().
				SetSort(bsonx.Doc{{provenDBVersionKey, bsonx.Int32(-1)}}).
				SetProjection(bsonx.Doc{
					{provenDBProofIDKey, bsonx.Int32(1)},
					{provenDBVersionKey, bsonx.Int32(1)},
					{provenDBSubmittedKey, bsonx.Int32(1)},
					{provenDBStatusKey, bsonx.Int32(1)},
				}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var result []verifiableVersion

	for cur.Next(ctx) {
		doc := bsonx.Doc{}
		err := cur.Decode(&doc)
		if err != nil {
			return nil, err
		}

		var vV verifiableVersion

		if p, ok := doc.Lookup(provenDBProofIDKey).StringValueOK(); ok {
			vV.proofID = p
		} else {
			continue
		}

		if v, ok := doc.Lookup(provenDBVersionKey).Int64OK(); ok {
			vV.versionID = v
		} else {
			continue
		}

		if t, ok := doc.Lookup(provenDBSubmittedKey).DateTimeOK(); ok {
			vV.submitTimestamp = time.Unix(0, t*int64(time.Millisecond))
		} else {
			continue
		}

		if s, ok := doc.Lookup(provenDBStatusKey).StringValueOK(); ok {
			vV.proofStatus = s
		} else {
			continue
		}

		result = append(result, vV)
	}

	return result, nil
}

func findDocs(ctx context.Context, collection *mongo.Collection, version int64, filter bsonx.Doc, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	opts = append(opts,
		options.Find().
			SetSort(bsonx.Doc{
				{provenDBDocMetaMinVersionKey, bsonx.Int32(1)},
				{provenDBDocMetaIDKey, bsonx.Int32(1)},
			}))

	andStatements := []bsonx.Val{
		bsonx.Document(bsonx.Doc{
			{
				provenDBDocMetaMinVersionKey,
				bsonx.Document(bsonx.Doc{
					{
						"$lte",
						bsonx.Int64(version),
					},
				}),
			},
		}),
		bsonx.Document(bsonx.Doc{
			{
				provenDBDocMetaMaxVersionKey,
				bsonx.Document(bsonx.Doc{
					{
						"$gte",
						bsonx.Int64(version),
					},
				}),
			},
		}),
	}
	if filter != nil {
		// If the filter contains $and array, append the elements to our $and array
		and, ok := filter.Lookup("$and").ArrayOK()
		if ok {
			for _, v := range and {
				andStatements = append(andStatements, v)
			}
			filter = filter.Delete("$and")
		}
		filter = filter.Append("$and", bsonx.Array(andStatements))
	} else {
		filter = bsonx.Doc{}.Append("$and", bsonx.Array(andStatements))
	}

	return collection.Find(ctx,
		filter,
		opts...,
	)
}

func getHashKeyFromDoc(doc bsonx.Doc) ([]byte, error) {
	v, err := doc.LookupErr(idKey)
	if err != nil {
		return nil, err
	}

	result, err := bsonx.Doc{
		{"", v},
	}.MarshalBSON()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func getIDValueFromHashKey(hashKey []byte) (bsonx.Val, error) {
	doc, err := bsonx.ReadDoc(hashKey)
	if err != nil {
		return bsonx.Null(), err
	}

	return doc.LookupErr("")
}

func getDocProofMap(ctx context.Context, database *mongo.Database, version int64, opt *docOpt, proofMap map[string]map[string]*merkle.Proof) (hash []byte, err error) {
	filter := bsonx.Doc{}
	err = bson.UnmarshalExtJSON([]byte(opt.docFilter), true, &filter)
	if err != nil {
		return nil, fmt.Errorf("invalid '--docFilter': %s", err)
	}

	var opts []*options.FindOptions

	if !opt.calcHash {
		opts = append(opts, options.Find().SetProjection(
			bsonx.Doc{
				{idKey, bsonx.Int32(0)},
				{provenDBDocMetaIDKey, bsonx.Int32(1)},
			},
		))
	}

	cur, err := findDocs(ctx, database.Collection(opt.colName), version, filter, opts...)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var docID []byte

	for cur.Next(ctx) {
		if len(docID) > 0 {
			return nil, fmt.Errorf("please make sure that '--collection' and '--docFilter' combined only returns a single document in version %v", version)
		}

		doc := bsonx.Doc{}
		err := cur.Decode(&doc)
		if err != nil {
			return nil, err
		}

		var metaDoc bsonx.Doc

		if opt.calcHash {
			hash, metaDoc, err = hashDocument(doc)
			if err != nil {
				return nil, err
			}
		} else {
			v, err := doc.LookupErr(provenDBDocMetaKey)
			if err != nil {
				return nil, err
			}

			md, ok := v.DocumentOK()
			if !ok {
				return nil, fmt.Errorf("%s is not a document", provenDBDocMetaKey)
			}

			metaDoc = md
		}

		b, err := getHashKeyFromDoc(metaDoc)
		if err != nil {
			return nil, err
		}

		docID = b
	}

	if len(docID) == 0 {
		return nil, fmt.Errorf("'--collection' and '--docFilter' combined doesn't return any document in version %v", version)
	}

	docProofMap, ok := proofMap[opt.colName]
	docProof := merkle.Proof{
		Key: docID,
	}

	if !ok {
		proofMap[opt.colName] = map[string]*merkle.Proof{
			string(docID): &docProof,
		}
	} else {
		docProofMap[string(docID)] = &docProof
	}

	return
}

// documentContainsPrefix checks the document elements to see if any elements that are documents or nested arrays
// contain fields that begin with the given prefix
func documentContainsPrefix(doc bsonx.Doc, prefix string) bool {
	for _, e := range doc {
		// Check top level fields
		if strings.HasPrefix(e.Key, prefix) {
			return true
		}
		// We need to see if the field contains a document type (could be an conditional operator)
		if e.Value.Type() == bsontype.EmbeddedDocument {
			if documentContainsPrefix(e.Value.Document(), prefix) {
				return true
			}
		}
		// If the field is an array, we also need to check this
		if e.Value.Type() == bsontype.Array {
			if arrayContainsPrefix(e.Value.Array(), prefix) {
				return true
			}
		}
	}
	return false
}

// arrayContainsPrefix checks the array elements to see if any elements that are documents or nested arrays
// contain fields that begin with the given prefix
func arrayContainsPrefix(arr bsonx.Arr, prefix string) bool {
	for _, v := range arr {
		if v.Type() == bsontype.EmbeddedDocument {
			if documentContainsPrefix(v.Document(), prefix) {
				return true
			}
		}
		if v.Type() == bsontype.Array {
			if arrayContainsPrefix(v.Array(), prefix) {
				return true
			}
		}
	}
	return false
}
