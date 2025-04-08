package btree

import "encoding/binary"

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
