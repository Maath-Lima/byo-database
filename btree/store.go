package btree

import (
	"os"
)

// | the_meta_page | pages... | root_node | pages... | (end_of_file)
// | root_ptr | page_used | ^ ^
// | | | |
// +----------|----------------------+ |
// | |
// +---------------------------------------+

// durability + atomicity
// ignoring concurrency and assume sequential access whitin 1 process
type KV struct {
	Path string   // file name
	fd   *os.File // follow how Windows OS deals with file access
	tree BTree
}

func (db *KV) Open() error //open or create

func (db *KV) Get(key []byte) ([]byte, bool) {
	return db.tree.Get(key)
}

func (db *KV) Set(key []byte, val []byte) error {
	db.tree.Insert(key, val)
	return updateFile(db)
}

func (db *KV) Del(key []byte) (bool, error) {
	deleted := db.tree.Delete(key)
	return deleted, updateFile(db)
}

// Just because nodes are written before the root doesn't mean the disk will persist them in that order due to OS caching, etc
func updateFile(db *KV) error {
	// 1. write new nodes
	if err := writePages(db); err != nil {
		return err
	}

	// 2. fsync to enforce the order between 1 and 3
	if err := db.fd.Sync(); err != nil {
		return err
	}

	// 3. update the root pointer atomically
	if err := updateRoot(db); err != nil {
		return err
	}

	// 4. fsync to make everything persistent
	return db.fd.Sync()
}
