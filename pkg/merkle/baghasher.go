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
 * @Date:   2018-07-31T14:38:36+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-04-02T13:25:22+11:00
 */

package merkle

import (
	"bytes"
	"fmt"

	hasher "github.com/SouthbankSoftware/provendb-verify/pkg/crypto/sha256"
)

// BagEntry is a key-value pair with first element as key and second as value
type BagEntry [2][]byte

// BagEntries is a slice of BagEntries
type BagEntries []BagEntry

// Len is the number of BagEntries
func (b BagEntries) Len() int {
	return len(b)
}

// Less reports whether the BagEntry with index i should sort before the BagEntry with index j
func (b BagEntries) Less(i, j int) bool {
	return bytes.Compare(b[i][0], b[j][0]) == -1
}

// Swap swaps the BagEntries with indexes i and j
func (b BagEntries) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// ValueHashAlgorithm represents value hash algorithm
type ValueHashAlgorithm string

// VHAS represents value hash algorithms
var VHAS = struct {
	None ValueHashAlgorithm
}{
	// None is the algorithm name used to hash BagEntry value by directly using value as the hash
	None: "none",
}

// HashCombiningAlgorithm represents hash combining algorithm
type HashCombiningAlgorithm string

// HCAS represents hash combining algorithms
var HCAS = struct {
	Sha256 HashCombiningAlgorithm
}{
	// None is the algorithm name used to combine two hashes using sha256
	Sha256: "sha256",
}

// PathNode represents a node along the merkle path
type PathNode struct {
	// LeftHash is the left hash value
	LeftHash []byte
	// RightHash is the right hash value
	RightHash []byte
}

// Proof represents the merkle proof of a key-value pair
type Proof struct {
	// Key is the key of the BagEntry this proof represents
	Key []byte
	// Value is the value of the BagEntry this proof represents
	Value []byte
	// RootHash is the merkle root hash
	RootHash []byte
	// ValueHashAlgorithm is the name of the value hash algorithm used
	ValueHashAlgorithm ValueHashAlgorithm
	// HashCombiningAlgorithm is the name of the hash combining used
	HashCombiningAlgorithm HashCombiningAlgorithm
	// Path is the merkle path
	Path []PathNode
	// Meta is any meta data associated with this proof
	Meta interface{}
}

// Verify verifies current Proof
func (p Proof) Verify() (verified bool, err error) {
	var hash []byte

	switch algo := p.ValueHashAlgorithm; algo {
	case VHAS.None:
		hash = p.Value
	default:
		return false, fmt.Errorf("%s is not a supported value hash algorithm", algo)
	}

	var hca func(bs ...[]byte) []byte

	switch algo := p.HashCombiningAlgorithm; algo {
	case HCAS.Sha256:
		hca = hasher.HashByteArray
	default:
		return false, fmt.Errorf("%s is not a supported hash combining algorithm", algo)
	}

	for _, pn := range p.Path {
		if len(pn.LeftHash) != 0 {
			hash = hca(pn.LeftHash, hash)
		} else {
			hash = hca(hash, pn.RightHash)
		}
	}

	if bytes.Compare(hash, p.RootHash) == 0 {
		return true, nil
	}

	return false, fmt.Errorf("recalculated root hash %x doesn't match hash %x in proof", hash, p.RootHash)
}

// BagHasher is the generic hashing interface for a bag of unordered key-value pairs
type BagHasher interface {
	// Patch patches the latest version of the hasher with BagEntries and optional keys for
	// generating proofs, and it returns the root hash and required proofs. When a value for a key
	// is empty, i.e. len([]byte) == 0, that key is used to delete any existing item with the same
	// key in the hasher. The order of the given entries will not affect the root hash if applying
	// different permutations of the same entries results in the same bag of BagEntries
	Patch(entries BagEntries, proofKeys ...[]byte) (hash []byte, proofs []Proof)

	// SaveVersion saves current version for the hasher and returns the next version number. Any
	// changes made to the saved version become finalized, and BagEntries of that version hence
	// become immutable. All future changes will be made to the next version
	SaveVersion() (nextVersion int64)

	// Version returns the current version number
	Version() (version int64)

	// GetProofs gets merkle proofs for BagEntries with given keys in given version
	GetProofs(version int64, keys ...[]byte) (proofs []Proof)

	// GetLatestProofs gets merkle proofs for BagEntries with given keys in the latest version
	GetLatestProofs(keys ...[]byte) (proofs []Proof)

	// Height returns the height of the underlie tree in BagHasher
	Height() (height int)

	// Size returns the number of nodes in the underlie tree in BagHasher
	Size() (size int)
}
