package main

import (
	"database/btree"
	"database/utils"
)

// | type | nkeys | pointers | offsets | key-values | unused |
// | 2B | 2B | nkeys * 8B | nkeys * 2B | ... | |
//
// | klen | vlen | key | val |
// | 2B | 2B | ... | ... |

// main
func main() {
	node1max := btree.HEADER + 8 + 2 + 4 + btree.BTREE_MAX_KEY_SIZE + btree.BTREE_MAX_VAL_SIZE
	utils.Assert(node1max <= btree.BTREE_PAGE_SIZE)

	// node := btree.BNode{
	// 	// HEADER (4 bytes)
	// 	0x02, 0x00, // Node Type (BNODE_LEAF = 2)
	// 	0x02, 0x00, // Key Count = 2

	// 	// NEXT LEAF POINTER (8 bytes)
	// 	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // No next leaf (NULL)
	// 	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // No next leaf (NULL)

	// 	// KV POSITION TABLE (4 bytes)
	// 	0x00, 0x00, // Offset of 1st KV Pair (points to position 24)
	// 	0x0C, 0x00, // Offset of 2nd KV Pair (points to position 36)

	// 	// FIRST KV PAIR (Key1: "ABC", Value1: "VAL1!")
	// 	0x03, 0x00, // Key1 Length = 3 bytes
	// 	0x05, 0x00, // Value1 Length = 5 bytes
	// 	'A', 'B', 'C', // Key1 = "ABC"
	// 	'V', 'A', 'L', '1', '!', // Value1 = "VAL1!"

	// 	// SECOND KV PAIR (Key2: "XYZ", Value2: "DATA2")
	// 	0x03, 0x00, // Key2 Length = 3 bytes
	// 	0x05, 0x00, // Value2 Length = 5 bytes
	// 	'X', 'Y', 'Z', // Key2 = "XYZ"
	// 	'D', 'A', 'T', 'A', '2', // Value2 = "DATA2"
	// }
}
