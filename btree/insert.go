package btree

import (
	"bytes"
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
		return 2, [3]BNode{left, right} // 2 nodes
	}

	leftleft := BNode(make([]byte, BTREE_PAGE_SIZE))
	middle := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(leftleft, middle, left)
	utils.Assert(leftleft.nbytes() <= BTREE_PAGE_SIZE)
	return 3, [3]BNode{leftleft, left, right} // 3 nodes
}

func nodeReplaceKidN(tree *BTree, new BNode, old BNode, idx uint16, kids ...BNode) {
	inc := uint16(len(kids))
	new.setHeader(BNODE_NODE, old.nkeys()+inc-1)
	nodeAppendRange(new, old, 0, 0, idx)
	for i, node := range kids {
		nodeAppendKV(new, idx+uint16(i), tree.new(node), node.getKey(0), nil)
	}
	nodeAppendRange(new, old, idx+inc, idx+1, old.nkeys()-(idx+1))
}

func treeInsert(tree *BTree, node BNode, key []byte, val []byte) BNode {
	// the extra size allows it to exceed 1 page temporarily
	// the result node it's allowed to be bigger than 1 page and will br split if so
	new := BNode(make([]byte, 2*BTREE_PAGE_SIZE))
	// where to inset the key?
	idx := nodeLookUpLE(node, key)
	switch node.btype() {
	case BNODE_LEAF:
		if bytes.Equal(key, node.getKey(idx)) {
			leafUpdate(new, node, idx, key, val) // update it
		} else {
			leafInsert(new, node, idx+1, key, val) // insert
		}
	case BNODE_NODE: // internal node, walk into the child node
		kptr := node.getPtr(idx)
		knode := treeInsert(tree, tree.get(kptr), key, val)
		// after insertion, split the result
		nsplit, split := nodeSplit3(knode)
		// deallocate the old kid node
		tree.del(kptr)
		// update the kid links
		nodeReplaceKidN(tree, new, node, idx, split[:nsplit]...)
	}

	return new
}

// insert a new key or update an existing key
func (tree *BTree) Insert(key []byte, val []byte) {
	// 1. check the length limit imposed by the node format
	utils.Assert(len(key) <= BTREE_MAX_KEY_SIZE)
	utils.Assert(len(val) <= BTREE_MAX_VAL_SIZE)

	if tree.root == 0 {
		root := BNode(make([]byte, BTREE_PAGE_SIZE))
		root.setHeader(BNODE_LEAF, 2)
		// dummy key for tree covering the whole key space
		nodeAppendKV(root, 0, 0, nil, nil) // sentinel value, to nodeLookupLE always find a position | used to remove an edge case
		nodeAppendKV(root, 1, 0, key, val)
		tree.root = tree.new(root)
		return
	}

	node := treeInsert(tree, tree.get(tree.root), key, val)
	nsplit, split := nodeSplit3(node)
	tree.del(tree.root)

	if nsplit > 1 {
		// add a new level
		root := BNode(make([]byte, BTREE_PAGE_SIZE))
		root.setHeader(BNODE_NODE, nsplit)
		for i, knode := range split[:nsplit] {
			ptr, key := tree.new(knode), knode.getKey(0)
			nodeAppendKV(root, uint16(i), ptr, key, nil)
		}
		tree.root = tree.new(root)
	} else {
		tree.root = tree.new(split[0])
	}
}
