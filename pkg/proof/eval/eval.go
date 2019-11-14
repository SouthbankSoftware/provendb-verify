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
 * @Date:   2018-08-22T13:22:09+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-11-14T14:55:27+11:00
 */

package eval

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"

	"github.com/SouthbankSoftware/provendb-verify/pkg/proof/queue"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
)

const (
	// SignaturePrefix is the prefix for the signature entry in a proof
	SignaturePrefix = "sig:"
)

// Eval evaluates given Proof JSON and calculate anchor infos such as merkle root
func Eval(proof interface{}) (result map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("failed to evaluate Proof: %s", r)
		}
	}()

	value := proof.(map[string]interface{})
	result = make(map[string]interface{})

	hash := value["hash"]

	result["hash"] = hash
	result["hash_id_node"] = value["hash_id_node"]
	result["hash_submitted_node_at"] = value["hash_submitted_node_at"]
	result["hash_id_core"] = value["hash_id_core"]
	result["hash_submitted_core_at"] = value["hash_submitted_core_at"]

	hashBA, err := hex.DecodeString(hash.(string))
	if err != nil {
		panic(err)
	}

	result["branches"] = evalBranches(hashBA, value["branches"].([]interface{}))

	return result, nil
}

// Branch evaluates a Chainpoint branch and returns the result branch and end hash
func Branch(startHash []byte, branch map[string]interface{}) (resultBranch map[string]interface{}, endHash []byte) {
	currHash := startHash
	resultBranch = make(map[string]interface{})

	var (
		resultAnchors []interface{}
		isBTC         bool
		btcQueue      *queue.Queue
	)

	if l := branch["label"]; l != nil {
		resultBranch["label"] = l

		if l == "btc_anchor_branch" {
			isBTC = true
			btcQueue = queue.New()
		}
	}

	checkSig := func(operand string) {
		if strings.HasPrefix(operand, SignaturePrefix) {
			resultBranch["sig"] = operand[len(SignaturePrefix):]
			resultBranch["sigHash"] = hex.EncodeToString(currHash)
		}
	}

	currBranchOps := branch["ops"].([]interface{})

	for j := 0; j < len(currBranchOps); j++ {
		currBranchOp := currBranchOps[j].(map[string]interface{})

		if r := currBranchOp["r"]; r != nil {
			op := r.(string)
			checkSig(op)
			currHash = append(currHash, str2ByteArray(op)...)
		} else if l := currBranchOp["l"]; l != nil {
			op := l.(string)
			checkSig(op)
			currHash = append(str2ByteArray(op), currHash...)
		} else if op := currBranchOp["op"]; op != nil {
			switch algo := op.(string); algo {
			case "sha-224":
				currHash = hashData(currHash, sha256.New224())
			case "sha-256":
				currHash = hashData(currHash, sha256.New())
			case "sha-384":
				currHash = hashData(currHash, sha512.New384())
			case "sha-512":
				currHash = hashData(currHash, sha512.New())
			case "sha3-224":
				currHash = hashData(currHash, sha3.New224())
			case "sha3-256":
				currHash = hashData(currHash, sha3.New256())
			case "sha3-384":
				currHash = hashData(currHash, sha3.New384())
			case "sha3-512":
				currHash = hashData(currHash, sha3.New512())
			case "sha-256-x2":
				hasher := sha256.New()
				currHash = hashData(currHash, hasher)
				hasher.Reset()
				currHash = hashData(currHash, hasher)

				if isBTC && btcQueue != nil {
					resultBranch["opReturnValue"] = hex.EncodeToString(btcQueue.Peek().([]byte))
					resultBranch["btcTxId"] = getReverseHexStr(currHash)

					btcQueue = nil
				}
			default:
				log.Warnf("The hashing algorithm %s is not supported", algo)
			}
		} else if anchors := currBranchOp["anchors"]; anchors != nil {
			resultAnchors = append(resultAnchors, evalAnchors(currHash, anchors.([]interface{}))...)
		}

		if isBTC && btcQueue != nil {
			// cache up to 3 hash results
			hash := make([]byte, len(currHash))
			copy(hash, currHash)
			btcQueue.Enqueue(hash)

			for btcQueue.Len() > 3 {
				btcQueue.Dequeue()
			}
		}
	}

	resultBranch["anchors"] = resultAnchors

	if branches := branch["branches"]; branches != nil {
		resultBranch["branches"] = evalBranches(currHash, branches.([]interface{}))
	}

	return resultBranch, currHash
}

func evalBranches(startHash []byte, branches []interface{}) (result []interface{}) {
	currHash := startHash

	for i := 0; i < len(branches); i++ {
		branch := branches[i].(map[string]interface{})

		resultBranch, endHash := Branch(currHash, branch)

		result = append(result, resultBranch)
		currHash = endHash
	}

	return result
}

func evalAnchors(currHash []byte, anchors []interface{}) (result []interface{}) {
	for i := 0; i < len(anchors); i++ {
		anchor := anchors[i].(map[string]interface{})

		resultAnchor := map[string]interface{}{
			"type":      anchor["type"],
			"anchor_id": anchor["anchor_id"],
		}

		if uris := anchor["uris"]; uris != nil {
			resultAnchor["uris"] = uris
		}

		if aType := anchor["type"]; aType == "btc" {
			// BTC merkle root values are in little endian byte order, which are different in
			// Chainpoint's big endian byte order
			resultAnchor["expected_value"] = getReverseHexStr(currHash)
		} else {
			resultAnchor["expected_value"] = hex.EncodeToString(currHash)
		}

		result = append(result, resultAnchor)
	}

	return result
}

func hashData(data []byte, hasher hash.Hash) []byte {
	hasher.Write(data)
	return hasher.Sum(nil)
}

// str2ByteArray converts a string to []byte by first treating the string as hex string then, if the
// conversion fails, utf8
func str2ByteArray(str string) []byte {
	result, err := hex.DecodeString(str)
	if err != nil {
		return []byte(str)
	}
	return result
}

func reverseClone(b []byte) []byte {
	bLen := len(b)
	result := make([]byte, bLen)

	for i := 0; i < bLen; i++ {
		result[bLen-i-1] = b[i]
	}

	return result
}

func getReverseHexStr(b []byte) string {
	return hex.EncodeToString(reverseClone(b))
}
