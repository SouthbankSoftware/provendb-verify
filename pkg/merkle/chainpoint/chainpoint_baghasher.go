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
 * @Last modified time: 2019-11-12T15:55:02+11:00
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
	Key       []byte
	Value     []byte
	Height    int8
	Size      int64
	Hash      []byte
	LeftHash  []byte
	RightHash []byte
	Parent    *Node
}

// BagHasher represents an instance of Chainpoint merkle tree based hasher
type BagHasher struct {
	Root *Node
}

// NewBagHasher returns a new Chainpoint merkle tree based hasher
func NewBagHasher() *BagHasher {
	return &BagHasher{
		Root: nil,
	}
}

func max(i, j int8) int8 {
	if i > j {
		return i
	}
	return j
}

// PairwiseCombine combines a layer of merkle nodes and returns one layer above the bottom layer
func PairwiseCombine(nodes []*Node) []*Node {
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
				Height:    1 + max(nodeJ.Height, nodeK.Height),
				Size:      1 + nodeJ.Size + nodeK.Size,
				Hash:      hasher.HashByteArray(nodeJ.Hash, nodeK.Hash),
				LeftHash:  nodeJ.Hash,
				RightHash: nodeK.Hash,
			}

			parent := resultNodes[i]
			nodeJ.Parent = parent
			nodeK.Parent = parent
		} else {
			resultNodes[i] = nodeJ
		}
	}

	return resultNodes
}

func calcPathToRoot(node *Node) []merkle.PathNode {
	var path []merkle.PathNode

	for node.Parent != nil {
		parent := node.Parent

		var pathNode merkle.PathNode

		if bytes.Compare(node.Hash, parent.LeftHash) == 0 {
			pathNode.RightHash = parent.RightHash
		} else {
			pathNode.LeftHash = parent.LeftHash
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
			Key:    key,
			Value:  entry[1],
			Height: 0,
			Size:   1,
			Hash:   entry[1],
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
		nodes = PairwiseCombine(nodes)
	}

	c.Root = nodes[0]
	hash = c.Root.Hash

	if len(proofNodes) > 0 {
		proofs = make([]merkle.Proof, len(proofNodes))
		for i, n := range proofNodes {
			proofs[i] = c.NodeToProof(n)
		}
	}

	return hash, proofs
}

// Create creates a Chainpoint merkle tree
func (c *BagHasher) Create(entries merkle.BagEntries) (
	hash []byte, leaves []*Node) {
	if len(entries) == 0 {
		return
	}

	buffer := make([]*Node, len(entries))
	leaves = make([]*Node, len(entries))

	for i, entry := range entries {
		key := entry[0]

		n := &Node{
			Key:    key,
			Value:  entry[1],
			Height: 0,
			Size:   1,
			Hash:   entry[1],
		}
		buffer[i] = n
		leaves[i] = n
	}

	for len(buffer) > 1 {
		buffer = PairwiseCombine(buffer)
	}

	c.Root = buffer[0]
	hash = c.Root.Hash

	return
}

// NodeToProof converts a merkle tree leaf node into a merkle proof
func (c *BagHasher) NodeToProof(node *Node) merkle.Proof {
	return merkle.Proof{
		Key:                    node.Key,
		Value:                  node.Value,
		RootHash:               c.Root.Hash,
		ValueHashAlgorithm:     merkle.VHAS.None,
		HashCombiningAlgorithm: merkle.HCAS.Sha256,
		Path:                   calcPathToRoot(node),
	}
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
	return int(c.Root.Height)
}

// Size returns the size of the Chainpoint merkle tree
func (c *BagHasher) Size() (size int) {
	return int(c.Root.Size)
}
