package btree

import (
	"database/utils"
	"encoding/binary"
)

// add a new node to a leaf node
// copy-on-write
func leafInsert(
	new BNode, old BNode, idx uint16,
	key []byte, val []byte,
) {
	new.setHeader(BNODE_LEAF, old.nkeys()+1)

	nodeAppendRange(new, old, 0, 0, idx)                   // copy the keys before `idx`
	nodeAppendKV(new, idx, 0, key, val)                    // the new key
	nodeAppendRange(new, old, idx+1, idx, old.nkeys()-idx) // keys from `idx`
}

// update a node if a key already exists, update only the value
func leafUpdate(
	new BNode, old BNode, idx uint16,
	key []byte, val []byte,
) {
	new.setHeader(BNODE_LEAF, old.nkeys()+1)

	nodeAppendRange(new, old, 0, 0, idx)                         // copy the keys before `idx`
	nodeAppendKV(new, idx, 0, key, val)                          // the new key
	nodeAppendRange(new, old, idx+1, idx+1, old.nkeys()-(idx+1)) // keys from `idx`
}

func nodeAppendRange(
	new BNode, old BNode, dstNew uint16, srcOld uint16, n uint16,
) {
	for i := uint16(0); i < n; i++ {
		dst, src := dstNew+i, srcOld+i
		nodeAppendKV(new, dst, old.getPtr(src), old.getKey(src), old.getVal(src))
	}
}

func nodeAppendKV(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	// ptrs
	new.setPtr(idx, ptr)

	//KVs
	pos := new.kvPos(idx)

	//4B KV sizes
	binary.LittleEndian.PutUint16(new[pos+0:], uint16(len(key)))
	binary.LittleEndian.PutUint16(new[pos+2:], uint16(len(val)))

	// KV data
	copy(new[pos+4:], key)
	copy(new[pos+4+uint16(len(key)):], val)

	// update the offset value for the next key
	new.setOffset(idx+1, new.getOffset(idx)+4+uint16(len(key)+len(val)))
}

// Split an oversized node into 2 nodes. The 2nd node always fits.
// ...we can use the half position as an initial guess, then move it left or right if the half is too large
func nodeSplit2(left BNode, right BNode, old BNode) {
	utils.Assert(old.nkeys() >= 2)

	// the initial guess
	nleft := old.nkeys() / 2
	// try to fit the left half
	left_bytes := func() uint16 {
		return 4 + 8*nleft + 2*nleft + old.getOffset(nleft)
	}
	for left_bytes() > BTREE_PAGE_SIZE {
		nleft--
	}
	utils.Assert(nleft >= 1)

	// try to fit the right half
	right_bytes := func() uint16 {
		return old.nbytes() - left_bytes() + 4
	}
	for right_bytes() > BTREE_PAGE_SIZE {
		nleft++
	}
	utils.Assert(nleft < old.nkeys())

	nright := old.nkeys() - nleft

	// new nodes
	left.setHeader(old.btype(), nleft)
	right.setHeader(old.btype(), nright)
	nodeAppendRange(left, old, 0, 0, nleft)
	nodeAppendRange(right, old, 0, nleft, nright)
	// NOTE: the left half may be still too big
	utils.Assert(right.nbytes() <= BTREE_PAGE_SIZE)
}

// split a node if it's too big. the results are 1~3 nodes.
func nodeSplit3(old BNode) (uint16, [3]BNode) {
	if old.nbytes() < BTREE_PAGE_SIZE {
		old = old[:BTREE_PAGE_SIZE]
		return 1, [3]BNode{old} // not split
	}

	left := BNode(make([]byte, 2*BTREE_PAGE_SIZE))
	right := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(left, right, old)
	if left.nbytes() <= BTREE_PAGE_SIZE {
		left = left[:BTREE_PAGE_SIZE]
		return 2, [3]BNode{left, right}
	}

	leftleft := BNode(make([]byte, BTREE_PAGE_SIZE))
	middle := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(leftleft, middle, left)
	utils.Assert(leftleft.nbytes() <= BTREE_PAGE_SIZE)
	return 3, [3]BNode{leftleft, left, right}
}
