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

	tree := btree.NewC()
	tree.Add("0", "first value")
	tree.Add("1", "second value")
	tree.Del("0")
	tree.PrintTree()
}
