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
 * @Date:   2018-08-02T09:41:56+10:00
 * @Last modified by:   guiguan
 * @Last modified time: 2019-08-26T16:29:25+10:00
 */

package chainpoint

import (
	"bytes"
	"math"

	hasher "github.com/SouthbankSoftware/provendb-verify/pkg/crypto/sha256"
	merkle "github.com/SouthbankSoftware/provendb-verify/pkg/merkle"
)

// Node represents a tree node in hasher
type Node struct {
	key       []byte
	value     []byte
	height    int8
	size      int64
	hash      []byte
	leftHash  []byte
	rightHash []byte
	parent    *Node
}

// BagHasher represents an instance of Chainpoint merkle tree based hasher
type BagHasher struct {
	root *Node
}

// NewBagHasher returns a new Chainpoint merkle tree based hasher
func NewBagHasher() *BagHasher {
	return &BagHasher{
		root: nil,
	}
}

func max(i, j int8) int8 {
	if i > j {
		return i
	}
	return j
}

func pairwiseCombine(nodes []*Node) []*Node {
	resultNodes := make([]*Node, int(math.Ceil(float64(len(nodes))/2.0)))

	for i := range resultNodes {
		var (
			j = 2 * i
			k = j + 1
		)

		nodeJ := nodes[j]

		if k < len(nodes) {
			nodeK := nodes[k]

			resultNodes[i] = &Node{
				height:    1 + max(nodeJ.height, nodeK.height),
				size:      1 + nodeJ.size + nodeK.size,
				hash:      hasher.HashByteArray(nodeJ.hash, nodeK.hash),
				leftHash:  nodeJ.hash,
				rightHash: nodeK.hash,
			}

			parent := resultNodes[i]
			nodeJ.parent = parent
			nodeK.parent = parent
		} else {
			resultNodes[i] = nodeJ
		}
	}

	return resultNodes
}

func calcPathToRoot(node *Node) []merkle.PathNode {
	var path []merkle.PathNode

	for node.parent != nil {
		parent := node.parent

		var pathNode merkle.PathNode

		if bytes.Compare(node.hash, parent.leftHash) == 0 {
			pathNode.RightHash = parent.rightHash
		} else {
			pathNode.LeftHash = parent.leftHash
		}

		path = append(path, pathNode)

		node = parent
	}

	return path
}

var keyExists = struct{}{}

// Patch initializes or reconstructs a Chainpoint merkle tree
func (c *BagHasher) Patch(entries merkle.BagEntries, proofKeys ...[]byte) (hash []byte, proofs []merkle.Proof) {
	if len(entries) == 0 {
		return
	}

	var (
		// struct{} doesn't take up space
		proofKeySet = make(map[string]struct{})
		proofNodes  []*Node
	)

	for _, p := range proofKeys {
		proofKeySet[string(p)] = keyExists
	}

	nodes := make([]*Node, len(entries))

	for i, entry := range entries {
		key := entry[0]

		nodes[i] = &Node{
			key:    key,
			value:  entry[1],
			height: 0,
			size:   1,
			hash:   entry[1],
		}

		if len(proofKeySet) != 0 {
			keyStr := string(key)
			if _, ok := proofKeySet[keyStr]; ok {
				proofNodes = append(proofNodes, nodes[i])
				delete(proofKeySet, keyStr)
			}
		}
	}

	for len(nodes) > 1 {
		nodes = pairwiseCombine(nodes)
	}

	c.root = nodes[0]
	hash = c.root.hash

	if len(proofNodes) > 0 {
		proofs = make([]merkle.Proof, len(proofNodes))
		for i, n := range proofNodes {
			proofs[i] = merkle.Proof{
				Key:                    n.key,
				Value:                  n.value,
				RootHash:               hash,
				ValueHashAlgorithm:     merkle.VHAS.None,
				HashCombiningAlgorithm: merkle.HCAS.Sha256,
				Path:                   calcPathToRoot(n),
			}
		}
	}

	return hash, proofs
}

// PatchWithFullProofs initializes or reconstructs a Chainpoint merkle tree and returns proofs for
// all entries
func (c *BagHasher) PatchWithFullProofs(entries merkle.BagEntries) (
	hash []byte, proofs []merkle.Proof) {
	if len(entries) == 0 {
		return
	}

	nodes := make([]*Node, len(entries))
	proofNodes := make([]*Node, len(entries))

	for i, entry := range entries {
		key := entry[0]

		n := &Node{
			key:    key,
			value:  entry[1],
			height: 0,
			size:   1,
			hash:   entry[1],
		}
		nodes[i] = n
		proofNodes[i] = n
	}

	for len(nodes) > 1 {
		nodes = pairwiseCombine(nodes)
	}

	c.root = nodes[0]
	hash = c.root.hash

	proofs = make([]merkle.Proof, len(proofNodes))
	for i, n := range proofNodes {
		proofs[i] = merkle.Proof{
			Key:                    n.key,
			Value:                  n.value,
			RootHash:               hash,
			ValueHashAlgorithm:     merkle.VHAS.None,
			HashCombiningAlgorithm: merkle.HCAS.Sha256,
			Path:                   calcPathToRoot(n),
		}
	}

	return hash, proofs
}

// SaveVersion is not implemented
func (c *BagHasher) SaveVersion() (nextVersion int64) {
	panic("Not implemented")
}

// Version is not implemented
func (c *BagHasher) Version() (version int64) {
	panic("Not implemented")
}

// GetProofs is not implemented
func (c *BagHasher) GetProofs(version int64, keys ...[]byte) (proofs []merkle.Proof) {
	panic("Not implemented")
}

// GetLatestProofs is not implemented
func (c *BagHasher) GetLatestProofs(keys ...[]byte) (proofs []merkle.Proof) {
	panic("Not implemented")
}

// Height returns the height of the Chainpoint merkle tree
func (c *BagHasher) Height() (height int) {
	return int(c.root.height)
}

// Size returns the size of the Chainpoint merkle tree
func (c *BagHasher) Size() (size int) {
	return int(c.root.size)
}
