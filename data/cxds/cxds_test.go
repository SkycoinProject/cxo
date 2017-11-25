package cxds

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/data/tests"
)

const testFileName = "test.db.goignore"

func testShouldNotPanic(t *testing.T) {
	if pc := recover(); pc != nil {
		t.Error("unexpected panic:", pc)
	}
}

func Test_one(t *testing.T) {
	if len(one) != 4 {
		t.Fatal("wrong length of the one")
	}
	if binary.BigEndian.Uint32(one) != 1 {
		t.Error("the one is not a one")
	}
}

func testDriveDS(t *testing.T) (ds data.CXDS) {
	var err error
	if ds, err = NewDriveCXDS(testFileName); err != nil {
		t.Fatal(err)
	}
	return
}

func TestNewDriveCXDS(t *testing.T) {
	// NewDriveCXDS(filePath string) (ds *DriveCXDS, err error)

	ds := testDriveDS(t)
	defer ds.Close()
}

func TestNewMemoryCXDS(t *testing.T) {
	// NewMemoryCXDS() (ds *MemoryCXDS, err error)

	ds := NewMemoryCXDS()
	defer ds.Close()
}

func TestCXDS_Get(t *testing.T) {
	// Get(key cipher.SHA256) (val []byte, rc uint32, err error)

	t.Run("memory", func(t *testing.T) {
		tests.CXDSGet(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		tests.CXDSGet(t, ds)
	})
}

func TestCXDS_Set(t *testing.T) {
	// Set(key cipher.SHA256, val []byte) (rc uint32, err error)

	t.Run("memory", func(t *testing.T) {
		tests.CXDSSet(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		tests.CXDSSet(t, ds)
	})
}

func TestCXDS_Inc(t *testing.T) {
	// Inc(key cipher.SHA256) (rc uint32, err error)

	t.Run("memory", func(t *testing.T) {
		tests.CXDSInc(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		tests.CXDSInc(t, ds)
	})
}

func TestCXDS_Close(t *testing.T) {
	// Close() (err error)

	t.Run("memory", func(t *testing.T) {
		tests.CXDSClose(t, NewMemoryCXDS())
	})

	t.Run("drive", func(t *testing.T) {
		ds := testDriveDS(t)
		defer os.Remove(testFileName)
		defer ds.Close()
		tests.CXDSClose(t, ds)
	})
}
