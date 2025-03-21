package main

import (
	"bytes"
	"encoding/binary"
)

// | type | nkeys | pointers | offsets | key-values | unused |
// | 2B | 2B | nkeys * 8B | nkeys * 2B | ... | |
//
// | klen | vlen | key | val |
// | 2B | 2B | ... | ... |

const HEADER = 4

const BTREE_PAGE_SIZE = 4096
const BTREE_MAX_KEY_SIZE = 1000
const BTREE_MAX_VAL_SIZE = 3000

const (
	BNODE_NODE = 1
	BNODE_LEAF = 2
)

type BNode []byte
type BTree struct {
	root uint64
	get  func(uint64) []byte
	new  func([]byte) uint64
	del  func(uint64)
}

func (node BNode) btype() uint16 {
	return binary.LittleEndian.Uint16(node[0:2])
}

func (node BNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:4])
}

func (node BNode) setHeader(btype uint16, nkeys uint16) {
	binary.LittleEndian.PutUint16(node[0:2], btype)
	binary.LittleEndian.PutUint16(node[2:4], nkeys)
}

// pointers
func (node BNode) getPtr(idx uint16) uint64 {
	assert(idx < node.nkeys())

	pos := HEADER + 8*idx
	return binary.LittleEndian.Uint64(node[pos:])
}

func (node BNode) setPtr(idx uint16, val uint64)

// offset list - helper
// ...offset list to locate the nth KV in O(1). This also allows binary searches within a node.
func offsetPos(node BNode, idx uint16) uint16 {
	assert(1 <= idx && idx <= node.nkeys())

	return HEADER + 8*node.nkeys() + 2*(idx-1)
}

func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0
	}

	return binary.LittleEndian.Uint16(node[offsetPos(node, idx):])
}

func (node BNode) setOffset(idx uint16, offset uint16)

// key-values
func (node BNode) kvPos(idx uint16) uint16 {
	assert(idx <= node.nkeys())

	return HEADER + 8*node.nkeys() + 2*node.nkeys() + node.getOffset(idx)
}

func (node BNode) getKey(idx uint16) []byte {
	assert(idx < node.nkeys())

	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])

	return node[pos+4:][:klen]
}

func (node BNode) getVal(idx uint16) []byte

func (node BNode) nbytes() uint16 {
	return node.kvPos(node.nkeys())
}

// lookup
// in a B+Tree, the first key in a leaf node is always a copy of the key from the parent node that separates it from its left sibling.
func nodeLookUpLE(node BNode, key []byte) uint16 {
	nkeys := node.nkeys()
	found := uint16(0)

	for i := uint16(1); i < nkeys; i++ {
		cmp := bytes.Compare(node.getKey(i), key)

		if cmp <= 0 {
			found = i
		}

		if cmp >= 0 {
			break
		}
	}

	return found
}

// main
func init() {
	node1max := HEADER + 8 + 2 + 4 + BTREE_MAX_KEY_SIZE + BTREE_MAX_VAL_SIZE
	assert(node1max <= BTREE_PAGE_SIZE)

	// node := BNode{
	// 	0x01, 0x02, 0x03, 0x04, // HEADER (4 bytes)
	// 	0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, // First pointer (8 bytes)
	// 	0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00, // Second pointer (8 bytes)
	// }

	// nodeLookUpLE(node)
}

func assert(condition bool) {
	if !condition {
		panic("Condition Failed")
	}
}
