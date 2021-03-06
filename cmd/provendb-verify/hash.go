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
 * @Date:   2019-04-02T13:42:23+11:00
 * @Last modified by:   Michael Harrison
 * @Last modified time: 2020-01-15T09:41:19+11:00
 */

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"sort"

	hasher "github.com/SouthbankSoftware/provendb-verify/pkg/crypto/sha256"
	"github.com/SouthbankSoftware/provendb-verify/pkg/merkle"
	"github.com/SouthbankSoftware/provendb-verify/pkg/merkle/chainpoint"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	log "github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

type hashResult struct {
	name   string
	hash   []byte
	height int
	size   int
	proofs []merkle.Proof
}

func hashDatabase(
	ctx context.Context,
	database *mongo.Database,
	version int64,
	proofMap map[string]map[string]*merkle.Proof,
	cols []string,
	ignoredCollections []string,
	filterStr string,
) (result hashResult, err error) {
	select {
	case <-ctx.Done():
		return result, ctx.Err()
	default:
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ignoredCollectionsRegex := ""
	for _, collection := range ignoredCollections {
		ignoredCollectionsRegex += "^" + collection + "$|"
	}

	colNameFilter := bsonx.Doc{
		{"$not", bsonx.Regex(
			"^"+provenDBMetaPrefix+"|^"+mongoDBSystemPrefix+"|"+ignoredCollectionsRegex+provenDBIgnoredSuffix+"$",
			"",
		)},
	}

	if len(cols) > 0 {
		vals := bsonx.Arr{}

		for _, colName := range cols {
			vals = append(vals, bsonx.String(colName))
		}

		colNameFilter = append(colNameFilter, bsonx.Elem{"$in", bsonx.Array(vals)})
	}

	// list normal collections
	cur, err := database.ListCollections(ctx,
		bsonx.Doc{
			{"name", bsonx.Document(colNameFilter)},
			{"type", bsonx.String("collection")},
		},
		options.ListCollections().SetNameOnly(true),
	)
	if err != nil {
		return result, err
	}
	defer cur.Close(ctx)

	colHashesCH := make(chan hashResult, 10)
	errCH := make(chan error)
	count := 0

	asyncHashCol := func(collection *mongo.Collection, proofKeys ...[]byte) {
		result, err := hashCollection(ctx, collection, version, filterStr, proofKeys...)
		if err != nil {
			select {
			case <-ctx.Done():
			case errCH <- err:
			}

			return
		}

		select {
		case <-ctx.Done():
		case colHashesCH <- result:
		}
	}

	for cur.Next(ctx) {
		doc := bsonx.Doc{}
		err := cur.Decode(&doc)
		if err != nil {
			return result, err
		}

		colName, ok := doc.Lookup("name").StringValueOK()
		if !ok {
			return result, fmt.Errorf("cannot get name when list collections")
		}

		var proofKeys [][]byte

		if docProofMap, ok := proofMap[colName]; ok {
			for _, v := range docProofMap {
				proofKeys = append(proofKeys, v.Key)
			}
		}

		go asyncHashCol(database.Collection(colName), proofKeys...)

		count++
	}

	var (
		entries      = make(merkle.BagEntries, 0, count)
		i            = 0
		height, size int
		progress     *mpb.Progress
		bar          *mpb.Bar
	)

	if !debug {
		progress = mpb.New(
			mpb.WithWidth(64),
			mpb.WithFormat(" \u2588\u2588\u2591 "),
			// canceled when context is canceled
			mpb.WithContext(ctx),
		)

		name := fmt.Sprintf("Hashing database `%s`...", database.Name())
		bar = progress.AddBar(int64(count+1),
			mpb.PrependDecorators(
				decor.Name(name, decor.WC{W: len(name) + 1, C: decor.DidentRight}),
				decor.CountersNoUnit("%d/%d"),
			),
			mpb.AppendDecorators(
				decor.OnComplete(decor.Percentage(), ""),
			),
			mpb.BarClearOnComplete(),
		)

		defer func() {
			// wait for progress bar to finish rendering
			progress.Wait()
		}()
	}

	for i < count {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case err := <-errCH:
			if !debug {
				// cancel progress bar render waiting
				progress.Abort(bar, false)
			}

			return result, err
		case r := <-colHashesCH:
			if r.hash == nil {
				// empty collection
				count--

				if !debug {
					bar.SetTotal(int64(count + 1), false)
				}

				// skip
				break
			}

			entries = append(entries, merkle.BagEntry{
				[]byte(r.name),
				r.hash,
			})

			if r.height > height {
				height = r.height
			}

			size += r.size

			if r.proofs != nil {
				if docProofMap, ok := proofMap[r.name]; ok {
					for _, p := range r.proofs {
						if proof, ok := docProofMap[string(p.Key)]; ok {
							proof.Value = p.Value
							proof.ValueHashAlgorithm = p.ValueHashAlgorithm
							proof.HashCombiningAlgorithm = p.HashCombiningAlgorithm
							proof.Path = p.Path
						}
					}
				}
			}

			if !debug {
				bar.IncrBy(1)
			}

			i++
		}
	}

	bagHasher := chainpoint.NewBagHasher()
	var colProofKeys [][]byte

	for k := range proofMap {
		colProofKeys = append(colProofKeys, []byte(k))
	}

	sort.Sort(entries)

	if debug {
		log.Debug("Hashes of collections will be assembled in this order:")

		for _, b := range entries {
			log.Debugf("\t%s", b[0])
		}
	}

	hash, proofs := bagHasher.Patch(entries, colProofKeys...)

	if !debug {
		bar.IncrBy(1)
	}

	if hash == nil {
		// empty database version
		return hashResult{
			hash: hasher.EmptyString,
		}, nil
	}

	var finalProofs []merkle.Proof

	// merge collection proofs with document proofs to form the complete proof for documents
	for _, p := range proofs {
		if docProofMap, ok := proofMap[string(p.Key)]; ok {
			for _, proof := range docProofMap {
				if len(proof.Value) != 0 {
					// merge for non-empty document proof
					proof.RootHash = hash
					proof.Path = append(proof.Path, p.Path...)
				}
			}
		}
	}

	for colName, docProofMap := range proofMap {
		for _, proof := range docProofMap {
			proof.Meta = colName
			finalProofs = append(finalProofs, *proof)
		}
	}

	return hashResult{
		database.Name(),
		hash,
		height + bagHasher.Height(),
		size - count + bagHasher.Size(),
		finalProofs,
	}, nil
}

func hashDocument(doc bsonx.Doc) (hash []byte, metaDoc bsonx.Doc, err error) {
	docID := doc.Lookup(idKey)
	idEl := bsonx.Elem{}

	defer func() {
		if err != nil {
			var pIDStr string

			if !idEl.Equal(bsonx.Elem{}) {
				pIDStr = fmt.Sprintf(" (ProvenDB ID `%v`)", idEl.Value.Interface())
			}

			err = fmt.Errorf("document `%v`%s: %s", docID, pIDStr, err)
		}
	}()

	metaDoc, ok := doc.Lookup(provenDBDocMetaKey).DocumentOK()
	if !ok {
		err = fmt.Errorf("cannot get ProvenDB metadata")
		return
	}

	doc = doc.Delete(idKey)

	idEl, err = metaDoc.LookupElementErr(idKey)
	if err != nil {
		return
	}

	minVerEl, err := metaDoc.LookupElementErr(provenDBMinVersionKey)
	if err != nil {
		return
	}

	getExpectedHash := func() (hash []byte, err error) {
		if el, er := metaDoc.LookupElementErr(provenDBHashKey); er == nil {
			if val := el.Value.Interface(); val != nil {
				if str, ok := val.(string); ok {
					if h, er := hex.DecodeString(str); er == nil {
						hash = h
					} else {
						err = er
						return
					}
				} else {
					err = fmt.Errorf("cannot convert %s element value to string", provenDBHashKey)
					return
				}
			}
		} else {
			err = er
			return
		}

		return
	}

	forgottenVl := metaDoc.Lookup(provenDBForgottenKey)
	if f, ok := forgottenVl.BooleanOK(); ok && f {
		hash, err = getExpectedHash()
		return
	}

	doc.Set(provenDBDocMetaKey, bsonx.Document(bsonx.Doc{
		idEl,
		minVerEl,
	}))

	docBA, err := doc.MarshalBSON()
	if err != nil {
		return
	}

	hash = hasher.HashByteArray(docBA)

	if !skipDocCheck {
		// check document hash against metadata
		expectedHash, er := getExpectedHash()
		if er != nil {
			err = er
			return
		}

		if expectedHash != nil && bytes.Compare(hash, expectedHash) != 0 {
			j, _ := bson.MarshalExtJSON(doc, true, false)
			err = fmt.Errorf("document hash mismatched. Expected: %x, actual: %x. Hashed document content: %s", expectedHash, hash, j)
			return
		}
	}

	return
}

func hashCollection(ctx context.Context, collection *mongo.Collection, version int64, filterStr string, proofKeys ...[]byte) (result hashResult, err error) {
	select {
	case <-ctx.Done():
		return result, ctx.Err()
	default:
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("collection `%s`: %s", collection.Name(), err)
		}
	}()

	// If there is a filter, unmarshal it.
	var filter bsonx.Doc
	if filterStr != "" {
		err = bson.UnmarshalExtJSON([]byte(filterStr), true, &filter)
		if err != nil {
			return result, fmt.Errorf("invalid `filter`: %s", err.Error())
		}
	}

	cur, err := findDocs(ctx, collection, version, filter)
	if err != nil {
		return result, err
	}
	defer cur.Close(ctx)

	var (
		entries = make(merkle.BagEntries, 0, 10)
		i       = 0
	)

	for cur.Next(ctx) {
		doc := bsonx.Doc{}
		err := cur.Decode(&doc)
		if err != nil {
			return result, err
		}

		hash, metaDoc, err := hashDocument(doc)
		if err != nil {
			return result, err
		}

		key, err := getHashKeyFromDoc(metaDoc)
		if err != nil {
			return result, err
		}

		entries = append(entries, merkle.BagEntry{
			key,
			hash,
		})
		i++
	}

	if i == 0 {
		// return empty string hash for empty collection
		return hashResult{
			collection.Name(),
			nil,
			0,
			0,
			nil,
		}, nil
	}

	bagHasher := chainpoint.NewBagHasher()

	hash, proofs := bagHasher.Patch(entries, proofKeys...)

	if debug {
		log.Debugf("Finished hashing collection `%s`: %x", collection.Name(), hash)
	}

	return hashResult{
		collection.Name(),
		hash,
		bagHasher.Height(),
		bagHasher.Size(),
		proofs,
	}, nil
}
