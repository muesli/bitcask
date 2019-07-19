package bitcask

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAll(t *testing.T) {
	var (
		db      *Bitcask
		testdir string
		err     error
	)

	assert := assert.New(t)

	testdir, err = ioutil.TempDir("", "bitcask")
	assert.NoError(err)

	t.Run("Open", func(t *testing.T) {
		db, err = Open(testdir)
		assert.NoError(err)
	})

	t.Run("Put", func(t *testing.T) {
		err = db.Put("foo", []byte("bar"))
		assert.NoError(err)
	})

	t.Run("Get", func(t *testing.T) {
		val, err := db.Get("foo")
		assert.NoError(err)
		assert.Equal([]byte("bar"), val)
	})

	t.Run("Len", func(t *testing.T) {
		assert.Equal(1, db.Len())
	})

	t.Run("Has", func(t *testing.T) {
		assert.True(db.Has("foo"))
	})

	t.Run("Keys", func(t *testing.T) {
		keys := make([]string, 0)
		for key := range db.Keys() {
			keys = append(keys, key)
		}
		assert.Equal([]string{"foo"}, keys)
	})

	t.Run("Fold", func(t *testing.T) {
		var (
			keys   []string
			values [][]byte
		)

		err := db.Fold(func(key string) error {
			value, err := db.Get(key)
			if err != nil {
				return err
			}
			keys = append(keys, key)
			values = append(values, value)
			return nil
		})
		assert.NoError(err)
		assert.Equal([]string{"foo"}, keys)
		assert.Equal([][]byte{[]byte("bar")}, values)
	})

	t.Run("Delete", func(t *testing.T) {
		err := db.Delete("foo")
		assert.NoError(err)
		_, err = db.Get("foo")
		assert.Error(err)
		assert.Equal(ErrKeyNotFound, err)
	})

	t.Run("Sync", func(t *testing.T) {
		err = db.Sync()
		assert.NoError(err)
	})

	t.Run("Close", func(t *testing.T) {
		err = db.Close()
		assert.NoError(err)
	})
}

func TestDeletedKeys(t *testing.T) {
	assert := assert.New(t)

	testdir, err := ioutil.TempDir("", "bitcask")
	assert.NoError(err)

	t.Run("Setup", func(t *testing.T) {
		var (
			db  *Bitcask
			err error
		)

		t.Run("Open", func(t *testing.T) {
			db, err = Open(testdir)
			assert.NoError(err)
		})

		t.Run("Put", func(t *testing.T) {
			err = db.Put("foo", []byte("bar"))
			assert.NoError(err)
		})

		t.Run("Get", func(t *testing.T) {
			val, err := db.Get("foo")
			assert.NoError(err)
			assert.Equal([]byte("bar"), val)
		})

		t.Run("Delete", func(t *testing.T) {
			err := db.Delete("foo")
			assert.NoError(err)
			_, err = db.Get("foo")
			assert.Error(err)
			assert.Equal(ErrKeyNotFound, err)
		})

		t.Run("Sync", func(t *testing.T) {
			err = db.Sync()
			assert.NoError(err)
		})

		t.Run("Close", func(t *testing.T) {
			err = db.Close()
			assert.NoError(err)
		})
	})

	t.Run("Reopen", func(t *testing.T) {
		var (
			db  *Bitcask
			err error
		)

		t.Run("Open", func(t *testing.T) {
			db, err = Open(testdir)
			assert.NoError(err)
		})

		t.Run("Get", func(t *testing.T) {
			_, err = db.Get("foo")
			assert.Error(err)
			assert.Equal(ErrKeyNotFound, err)
		})

		t.Run("Close", func(t *testing.T) {
			err = db.Close()
			assert.NoError(err)
		})
	})
}

func TestMaxKeySize(t *testing.T) {
	assert := assert.New(t)

	testdir, err := ioutil.TempDir("", "bitcask")
	assert.NoError(err)

	var db *Bitcask

	t.Run("Open", func(t *testing.T) {
		db, err = Open(testdir, WithMaxKeySize(16))
		assert.NoError(err)
	})

	t.Run("Put", func(t *testing.T) {
		key := strings.Repeat(" ", 17)
		value := []byte("foobar")
		err = db.Put(key, value)
		assert.Error(err)
		assert.Equal(ErrKeyTooLarge, err)
	})
}

func TestMaxValueSize(t *testing.T) {
	assert := assert.New(t)

	testdir, err := ioutil.TempDir("", "bitcask")
	assert.NoError(err)

	var db *Bitcask

	t.Run("Open", func(t *testing.T) {
		db, err = Open(testdir, WithMaxValueSize(16))
		assert.NoError(err)
	})

	t.Run("Put", func(t *testing.T) {
		key := "foo"
		value := []byte(strings.Repeat(" ", 17))
		err = db.Put(key, value)
		assert.Error(err)
		assert.Equal(ErrValueTooLarge, err)
	})
}

func TestOpenMerge(t *testing.T) {
	assert := assert.New(t)

	testdir, err := ioutil.TempDir("", "bitcask")
	assert.NoError(err)

	t.Run("Setup", func(t *testing.T) {
		var (
			db  *Bitcask
			err error
		)

		t.Run("Open", func(t *testing.T) {
			db, err = Open(testdir, WithMaxDatafileSize(32))
			assert.NoError(err)
		})

		t.Run("Put", func(t *testing.T) {
			for i := 0; i < 1024; i++ {
				err = db.Put(string(i), []byte(strings.Repeat(" ", 1024)))
				assert.NoError(err)
			}
		})

		t.Run("Get", func(t *testing.T) {
			for i := 0; i < 32; i++ {
				err = db.Put(string(i), []byte(strings.Repeat(" ", 1024)))
				assert.NoError(err)
				val, err := db.Get(string(i))
				assert.NoError(err)
				assert.Equal([]byte(strings.Repeat(" ", 1024)), val)
			}
		})

		t.Run("Sync", func(t *testing.T) {
			err = db.Sync()
			assert.NoError(err)
		})

		t.Run("Close", func(t *testing.T) {
			err = db.Close()
			assert.NoError(err)
		})
	})

	t.Run("Merge", func(t *testing.T) {
		var (
			db  *Bitcask
			err error
		)

		t.Run("Open", func(t *testing.T) {
			db, err = Open(testdir)
			assert.NoError(err)
		})

		t.Run("Get", func(t *testing.T) {
			for i := 0; i < 32; i++ {
				val, err := db.Get(string(i))
				assert.NoError(err)
				assert.Equal([]byte(strings.Repeat(" ", 1024)), val)
			}
		})

		t.Run("Close", func(t *testing.T) {
			err = db.Close()
			assert.NoError(err)
		})
	})
}

func TestMergeOpen(t *testing.T) {
	var (
		db  *Bitcask
		err error
	)

	assert := assert.New(t)

	testdir, err := ioutil.TempDir("", "bitcask")
	assert.NoError(err)

	t.Run("Setup", func(t *testing.T) {
		t.Run("Open", func(t *testing.T) {
			db, err = Open(testdir, WithMaxDatafileSize(32))
			assert.NoError(err)
		})

		t.Run("Put", func(t *testing.T) {
			for i := 0; i < 1024; i++ {
				err = db.Put(string(i), []byte(strings.Repeat(" ", 1024)))
				assert.NoError(err)
			}
		})

		t.Run("Get", func(t *testing.T) {
			for i := 0; i < 32; i++ {
				err = db.Put(string(i), []byte(strings.Repeat(" ", 1024)))
				assert.NoError(err)
				val, err := db.Get(string(i))
				assert.NoError(err)
				assert.Equal([]byte(strings.Repeat(" ", 1024)), val)
			}
		})

		t.Run("Sync", func(t *testing.T) {
			err = db.Sync()
			assert.NoError(err)
		})

		t.Run("Close", func(t *testing.T) {
			err = db.Close()
			assert.NoError(err)
		})
	})

	t.Run("Merge", func(t *testing.T) {
		t.Run("Merge", func(t *testing.T) {
			err = Merge(testdir, true)
			assert.NoError(err)
		})

		t.Run("Open", func(t *testing.T) {
			db, err = Open(testdir)
			assert.NoError(err)
		})

		t.Run("Get", func(t *testing.T) {
			for i := 0; i < 32; i++ {
				val, err := db.Get(string(i))
				assert.NoError(err)
				assert.Equal([]byte(strings.Repeat(" ", 1024)), val)
			}
		})

		t.Run("Close", func(t *testing.T) {
			err = db.Close()
			assert.NoError(err)
		})
	})
}

func TestConcurrent(t *testing.T) {
	var (
		db  *Bitcask
		err error
	)

	assert := assert.New(t)

	testdir, err := ioutil.TempDir("", "bitcask")
	assert.NoError(err)

	t.Run("Setup", func(t *testing.T) {
		t.Run("Open", func(t *testing.T) {
			db, err = Open(testdir)
			assert.NoError(err)
		})

		t.Run("Put", func(t *testing.T) {
			err = db.Put("foo", []byte("bar"))
			assert.NoError(err)
		})
	})

	t.Run("Concurrent", func(t *testing.T) {
		t.Run("Put", func(t *testing.T) {
			f := func(wg *sync.WaitGroup, x int) {
				defer func() {
					wg.Done()
				}()
				for i := 0; i <= 100; i++ {
					if i%x == 0 {
						key := fmt.Sprintf("k%d", i)
						value := []byte(fmt.Sprintf("v%d", i))
						err := db.Put(key, value)
						assert.NoError(err)
					}
				}
			}

			wg := &sync.WaitGroup{}

			go f(wg, 2)
			go f(wg, 3)
			go f(wg, 5)
			wg.Add(3)

			wg.Wait()
		})

		t.Run("Get", func(t *testing.T) {
			f := func(wg *sync.WaitGroup, N int) {
				defer func() {
					wg.Done()
				}()
				for i := 0; i <= N; i++ {
					value, err := db.Get("foo")
					assert.NoError(err)
					assert.Equal([]byte("bar"), value)
				}
			}

			wg := &sync.WaitGroup{}

			go f(wg, 100)
			go f(wg, 100)
			go f(wg, 100)
			wg.Add(3)

			wg.Wait()
		})

		t.Run("Close", func(t *testing.T) {
			err = db.Close()
			assert.NoError(err)
		})
	})
}

func TestScan(t *testing.T) {
	assert := assert.New(t)

	testdir, err := ioutil.TempDir("", "bitcask")
	assert.NoError(err)

	var db *Bitcask

	t.Run("Setup", func(t *testing.T) {
		t.Run("Open", func(t *testing.T) {
			db, err = Open(testdir)
			assert.NoError(err)
		})

		t.Run("Put", func(t *testing.T) {
			var items = map[string][]byte{
				"1":     []byte("1"),
				"2":     []byte("2"),
				"3":     []byte("3"),
				"food":  []byte("pizza"),
				"foo":   []byte("foo"),
				"fooz":  []byte("fooz ball"),
				"hello": []byte("world"),
			}
			for k, v := range items {
				err = db.Put(k, v)
				assert.NoError(err)
			}
		})
	})

	t.Run("Scan", func(t *testing.T) {
		var (
			vals     []string
			expected = []string{
				"foo",
				"fooz ball",
				"pizza",
			}
		)

		err = db.Scan("fo", func(key string) error {
			val, err := db.Get(key)
			assert.NoError(err)
			vals = append(vals, string(val))
			return nil
		})
		sort.Strings(vals)
		assert.Equal(expected, vals)
	})
}

func TestLocking(t *testing.T) {
	assert := assert.New(t)

	testdir, err := ioutil.TempDir("", "bitcask")
	assert.NoError(err)

	db, err := Open(testdir)
	assert.NoError(err)
	defer db.Close()

	_, err = Open(testdir)
	assert.Error(err)
	assert.Equal(ErrDatabaseLocked, err)
}

type benchmarkTestCase struct {
	name string
	size int
}

func BenchmarkGet(b *testing.B) {
	testdir, err := ioutil.TempDir("", "bitcask")
	if err != nil {
		b.Fatal(err)
	}

	db, err := Open(testdir)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	tests := []benchmarkTestCase{
		{"128B", 128},
		{"256B", 256},
		{"512B", 512},
		{"1K", 1024},
		{"2K", 2048},
		{"4K", 4096},
		{"8K", 8192},
		{"16K", 16384},
		{"32K", 32768},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			key := "foo"
			value := []byte(strings.Repeat(" ", tt.size))

			err = db.Put(key, value)
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				val, err := db.Get(key)
				if err != nil {
					b.Fatal(err)
				}
				if string(val) != string(value) {
					b.Errorf("unexpected value")
				}
			}
		})
	}
}

func BenchmarkPut(b *testing.B) {
	testdir, err := ioutil.TempDir("", "bitcask")
	if err != nil {
		b.Fatal(err)
	}

	db, err := Open(testdir)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	tests := []benchmarkTestCase{
		{"128B", 128},
		{"256B", 256},
		{"512B", 512},
		{"1K", 1024},
		{"2K", 2048},
		{"4K", 4096},
		{"8K", 8192},
		{"16K", 16384},
		{"32K", 32768},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			key := "foo"
			value := []byte(strings.Repeat(" ", tt.size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := db.Put(key, value)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkScan(b *testing.B) {
	testdir, err := ioutil.TempDir("", "bitcask")
	if err != nil {
		b.Fatal(err)
	}

	db, err := Open(testdir)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	var items = map[string][]byte{
		"1":     []byte("1"),
		"2":     []byte("2"),
		"3":     []byte("3"),
		"food":  []byte("pizza"),
		"foo":   []byte("foo"),
		"fooz":  []byte("fooz ball"),
		"hello": []byte("world"),
	}
	for k, v := range items {
		err := db.Put(k, v)
		if err != nil {
			b.Fatal(err)
		}
	}

	var expected = []string{"foo", "food", "fooz"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var keys []string
		err = db.Scan("fo", func(key string) error {
			keys = append(keys, key)
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
		sort.Strings(keys)
		if !reflect.DeepEqual(expected, keys) {
			b.Fatal(fmt.Errorf("expected keys=#%v got=%#v", expected, keys))
		}
	}
}
