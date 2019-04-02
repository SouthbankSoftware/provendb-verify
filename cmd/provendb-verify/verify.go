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
 * @Date:   2019-04-02T13:37:34+11:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-04-02T13:38:07+11:00
 */

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/SouthbankSoftware/provendb-verify/pkg/merkle"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/anchor"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/eval"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/schema"
	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/status"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

func verifyProofArchive(ctx context.Context, filename string) (msg string, er error) {
	fmt.Printf("Loading ProvenDB Proof Archive `%s`...\n", filename)

	defer func() {
		if r := recover(); r != nil {
			er = r.(error)
		}

		prefix := "ProvenDB Proof Archive"

		if er != nil {
			msg = prefix + " is falsified"
		} else {
			msg = prefix + " is verified"
		}
	}()

	r, err := zip.OpenReader(filename)
	if err != nil {
		er = err
		return
	}
	defer r.Close()

	var (
		doc   bsonx.Doc
		proof interface{}
	)

	getData := func(f *zip.File) (data []byte, err error) {
		rc, err := f.Open()
		if err != nil {
			return
		}
		defer rc.Close()

		return ioutil.ReadAll(rc)
	}

	for _, f := range r.File {
		// The __MACOSX folder is created when a Mac user creates and archive (also called a zip
		// file) using the Mac
		if f.Mode().IsRegular() && !strings.HasPrefix(f.Name, "__MACOSX") {
			if strings.HasSuffix(f.Name, ".proof.json") {
				data, err := getData(f)
				if err != nil {
					er = err
					return
				}

				err = json.Unmarshal(data, &proof)
				if err != nil {
					er = err
					return
				}
			} else if strings.HasSuffix(f.Name, ".doc.json") {
				data, err := getData(f)
				if err != nil {
					er = err
					return
				}

				err = bson.UnmarshalExtJSON(data, true, &doc)
				if err != nil {
					er = err
					return
				}
			}
		}
	}

	if len(doc) == 0 {
		er = fmt.Errorf("`.doc.json` is missing from the archive")
		return
	}

	if proof == nil {
		er = fmt.Errorf("`.proof.json` is missing from the archive")
		return
	}

	err = schema.Verify(proof)
	if err != nil {
		er = err
		return
	}

	expectedHash, err := hex.DecodeString(proof.(map[string]interface{})["hash"].(string))
	if err != nil {
		er = err
		return
	}

	actualHash, _, err := hashDocument(doc)
	if err != nil {
		er = err
		return
	}

	if bytes.Compare(actualHash, expectedHash) != 0 {
		er = fmt.Errorf("document hash mismatched. Expected: %x, actual: %x", expectedHash, actualHash)
		return
	}

	fmt.Println("Verifying Chainpoint Proof...")

	evaluatedProof, err := eval.Eval(proof)
	if err != nil {
		er = err
		return
	}

	err = anchor.Verify(ctx, evaluatedProof)
	if err != nil {
		er = err
		return
	}

	return
}

func verifyProof(ctx context.Context, database *mongo.Database, proof interface{}, version int64, opts ...interface{}) (msg string, err error) {
	var (
		inProofType, outProofType proofType
		proofName, outPath        string
		proofDocOpt               *docOpt
	)

	if database != nil {
		outProofType = proofTypes.database
		proofName = "`" + database.Name() + "`"
	} else {
		outProofType = proofTypes.raw
	}

	for _, opt := range opts {
		switch o := opt.(type) {
		case outOpt:
			outPath = o.path
		case docOpt:
			if database != nil {
				outProofType = proofTypes.document
				proofName = fmt.Sprintf("in `%s` with filter `%s`", o.colName, o.docFilter)
				proofDocOpt = &o
			}
		}
	}

	defer func() {
		if r := recover(); r != nil {
			err = status.NewVerificationStatusError(status.VerificationStatusFalsified, r.(error))
		}

		var prefix string

		if outProofType == proofTypes.raw {
			prefix = "Chainpoint Proof"
		} else {
			prefix = fmt.Sprintf("%s%s %s in version %v", strings.ToUpper(string(outProofType[:1])), outProofType[1:], proofName, version)
		}

		if err != nil {
			msg = "unable to verify " + prefix

			if se, ok := err.(*status.VerificationStatusError); ok {
				if se.Status == status.VerificationStatusFalsified {
					msg = prefix + " is falsified"
				}
			}
		} else {
			msg = prefix + " is verified"
		}
	}()

	err = schema.Verify(proof)
	if err != nil {
		err = status.NewVerificationStatusError(status.VerificationStatusFalsified, err)
		return
	}

	inProofType, err = getProofType(proof)
	if err != nil {
		err = status.NewVerificationStatusError(status.VerificationStatusFalsified, err)
		return
	}

	var hr hashResult

	if database != nil {
		var (
			proofMap     map[string]map[string]*merkle.Proof
			expectedHash []byte
			actualHash   []byte
		)

		if proofDocOpt != nil {
			if inProofType == proofTypes.document {
				proofDocOpt.calcHash = true
			}

			proofMap = make(map[string]map[string]*merkle.Proof)
			var hash []byte
			hash, err = getDocProofMap(ctx, database, version, proofDocOpt, proofMap)
			if err != nil {
				return
			}

			if inProofType == proofTypes.document {
				// we are verifying a document against a document Chainpoint Proof, no need to
				// reconstruct database merkle tree
				actualHash = hash
			}
		} else if inProofType == proofTypes.document {
			// we are verifying a database against a document Chainpoint Proof, we need to convert
			// the Proof to database Proof
			proof, err = docProof2DBProof(proof)
			if err != nil {
				return
			}
		}

		expectedHash, err = hex.DecodeString(proof.(map[string]interface{})["hash"].(string))
		if err != nil {
			err = status.NewVerificationStatusError(status.VerificationStatusFalsified, err)
			return
		}

		if actualHash == nil {
			hr, err = hashDatabase(ctx, database, version, proofMap)
			if err != nil {
				return
			}

			actualHash = hr.hash
		}

		if bytes.Compare(actualHash, expectedHash) != 0 {
			var prefix string

			if outProofType == proofTypes.database {
				prefix = "database merkle root"
			} else {
				prefix = "document"
			}

			err = status.NewVerificationStatusError(status.VerificationStatusFalsified, fmt.Errorf("%s hash mismatched. Expected: %x, actual: %x", prefix, expectedHash, actualHash))
			return
		}
	}

	fmt.Println("Verifying Chainpoint Proof...")

	evaluatedProof, err := eval.Eval(proof)
	if err != nil {
		err = status.NewVerificationStatusError(status.VerificationStatusFalsified, err)
		return
	}

	err = anchor.Verify(ctx, evaluatedProof)
	if err != nil {
		return
	}

	if outPath != "" {
		fmt.Printf("Outputting %s Chainpoint Proof to `%s`...\n", outProofType, outPath)

		if inProofType == proofTypes.database && outProofType == proofTypes.document && len(hr.proofs) > 0 {
			// convert database Proof to document Proof by embedding document merkle path
			proof, err = dbProof2DocProof(proof, hr.proofs[0])
			if err != nil {
				return
			}
		}

		err = saveProof(outPath, proof)
		if err != nil {
			return
		}
	}

	return
}
