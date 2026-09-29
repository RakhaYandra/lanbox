package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRecordListRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Record(Entry{Name: "a.zip", Kind: "upload", Size: 10, Millis: 5, SHA256: "abc", Status: "completed", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := db.Record(Entry{Name: "b.zip", Kind: "download", Size: 20, Status: "cancelled", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	got, err := db.List(20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "b.zip" || got[1].SHA256 != "abc" {
		t.Errorf("list = %+v", got)
	}
	page, err := db.List(1, 1)
	if err != nil || len(page) != 1 || page[0].Name != "a.zip" {
		t.Errorf("page = %+v, err = %v", page, err)
	}
}

func TestRetentionPrune(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	old := time.Now().Add(-100 * 24 * time.Hour)
	if err := db.Record(Entry{Name: "old", Kind: "upload", CreatedAt: old}); err != nil {
		t.Fatal(err)
	}
	if err := db.Record(Entry{Name: "new", Kind: "upload", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	got, err := db.List(20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "new" {
		t.Errorf("retention = %+v", got)
	}
}
