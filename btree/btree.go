package btree

import (
	"bytes"
	"database/utils"
	"encoding/binary"
	"fmt"
	"unsafe"
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
	// root pointer (a nonzero page number)
	root uint64
	// callbacks for managing on-disk pages
	get func(uint64) []byte // read data from a page number
	new func([]byte) uint64 // allocate a new page number with data
	del func(uint64)        // deallocate a page number
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

func (node BNode) setPtr(idx uint16, val uint64) {
	utils.Assert(idx < node.nkeys())

	pos := HEADER + 8*idx
	binary.LittleEndian.PutUint64(node[pos:], val)
}

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
// returns position of kv-pair at given index
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

func (node BNode) getVal(idx uint16) []byte {
	utils.Assert(idx < node.nkeys())
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos+0:])
	vlen := binary.LittleEndian.Uint16(node[pos+2:])

	return node[pos+4+klen:][:vlen]
}

// return node size in bytes
func (node BNode) nbytes() uint16 {
	return node.kvPos(node.nkeys())
}

// lookup
// in a B+Tree, the first key in a leaf node is always a copy of the key from the parent node that separates it from its left sibling.
// if found = 0 the search key is smaller than any real key in this node, but we're in the correct node because the parent guided us here.
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

type C struct {
	tree  BTree
	ref   map[string]string // ref for data
	pages map[uint64]BNode  // in-memory pages
}

func NewC() *C {
	pages := map[uint64]BNode{}
	return &C{
		tree: BTree{
			get: func(ptr uint64) []byte {
				node, ok := pages[ptr]
				utils.Assert(ok)
				return node
			},
			new: func(node []byte) uint64 {
				utils.Assert(BNode(node).nbytes() <= BTREE_PAGE_SIZE)
				ptr := uint64(uintptr(unsafe.Pointer(&node[0])))
				utils.Assert(pages[ptr] == nil)
				pages[ptr] = node
				return ptr
			},
			del: func(ptr uint64) {
				utils.Assert(pages[ptr] != nil)
				delete(pages, ptr)
			},
		},
		ref:   map[string]string{},
		pages: pages,
	}
}

func (c *C) Add(key string, val string) {
	c.tree.Insert([]byte(key), []byte(val))
	c.ref[key] = val
}

func (c *C) Del(key string) bool {
	delete(c.ref, key)
	return c.tree.Delete([]byte(key))
}

func (c *C) PrintTree() {
	fmt.Println("Pages:")
	for pt, node := range c.pages {
		fmt.Println("Pointer:", pt)
		fmt.Println("BNode data:", node)
	}
}
