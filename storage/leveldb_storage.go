package storage

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type LevelDBStorage struct {
	db *leveldb.DB
}

func NewLevelDBStorage(path string) (*LevelDBStorage, error) {
	db, err := leveldb.OpenFile(path, &opt.Options{
		OpenFilesCacheCapacity: 500,
		BlockCacheCapacity:     8 * opt.MiB,
		BlockSize:              4 * opt.MiB,
		Filter:                 filter.NewBloomFilter(10),
	})

	if err != nil {
		return nil, err
	}

	return &LevelDBStorage{
		db: db,
	}, nil
}

// Get return value to the key in Storage
func (storage *LevelDBStorage) Get(key []byte) ([]byte, error) {
	value, err := storage.db.Get(key, nil)
	if err != nil && err == leveldb.ErrNotFound {
		return nil, ErrKeyNotFound
	}

	return value, err
}

// Put put the key-value entry to Storage
func (storage *LevelDBStorage) Put(key []byte, value []byte) error {
	return storage.db.Put(key, value, nil)
}

func (storage *LevelDBStorage) Del(key []byte) error {
	return storage.db.Delete(key, nil)
}

// EnableBatch enable batch write.
func (db *LevelDBStorage) EnableBatch() {
}

// Flush write and flush pending batch write.
func (db *LevelDBStorage) Flush() error {
	return nil
}

// DisableBatch disable batch write.
func (db *LevelDBStorage) DisableBatch() {
}
