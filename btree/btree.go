package btree

import (
	"bytes"
	"database/utils"
	"encoding/binary"
)

const HEADER = 4

const BTREE_PAGE_SIZE = 4096
const BTREE_MAX_KEY_SIZE = 1000
const BTREE_MAX_VAL_SIZE = 3000

const (
	BNODE_NODE = 1
	BNODE_LEAF = 2
)

// node structure
// | type | nkeys | pointers | offsets | key-values | unused |
// | 2B | 2B | nkeys * 8B | nkeys * 2B | ... | |
//
// | klen | vlen | key | val |
// | 2B | 2B | ... | ... |
type BNode []byte

type BTree struct {
	root uint64
	get  func(uint64) []byte
	new  func([]byte) uint64
	del  func(uint64)
}

// header
// returns node type
func (node BNode) btype() uint16 {
	return binary.LittleEndian.Uint16(node[0:2])
}

// return node number of keys
func (node BNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:4])
}

func (node BNode) setHeader(btype uint16, nkeys uint16) {
	binary.LittleEndian.PutUint16(node[0:2], btype)
	binary.LittleEndian.PutUint16(node[2:4], nkeys)
}

// pointers
// return pointer to child node
func (node BNode) getPtr(idx uint16) uint64 {
	utils.Assert(idx < node.nkeys())

	pos := HEADER + 8*idx
	return binary.LittleEndian.Uint64(node[pos:])
}

func (node BNode) setPtr(idx uint16, val uint64) {}

// offset - helper to find KVs
// ...offset list to locate the nth KV in O(1). This also allows binary searches within a node.
// returns the location of the kv pair at given index
func offsetPos(node BNode, idx uint16) uint16 {
	utils.Assert(1 <= idx && idx <= node.nkeys())

	return HEADER + 8*node.nkeys() + 2*(idx-1)
}

func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0
	}

	return binary.LittleEndian.Uint16(node[offsetPos(node, idx):])
}

func (node BNode) setOffset(idx uint16, offset uint16) {}

// key-values
// returns position of kv-pair at given idenx
func (node BNode) kvPos(idx uint16) uint16 {
	utils.Assert(idx <= node.nkeys())

	base := HEADER + 8*node.nkeys() + 2*node.nkeys()
	offset := node.getOffset(idx)

	return base + offset
}

// returns key of key-pair
func (node BNode) getKey(idx uint16) []byte {
	utils.Assert(idx < node.nkeys())

	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])

	return node[pos+4:][:klen]
}

// func (node BNode) getVal(idx uint16) []byte

// return node size in bytes
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

// add a new node to a leaf node
// copy-on-write
// func leafInsert(
// 	new BNode, old BNode, idx uint16,
// 	key []byte, val []byte,
// ) {
// 	new.setHeader(BNODE_LEAF, old.nkeys()+1)

// 	nodeAppendRange(new, old, 0, 0, idx)
// 	nodeAppendKV(new, idx, 0, key, val)
// 	nodeAppendRange(new, old, idx+1, idx, old.nkeys()-idx)
// }

// func nodeAppendKV(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
// 	// ptrs
// 	new.setPtr()
// }
