package persistence

import (
	"testing"
	"encoding/hex"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"net/url"
	"time"
)

const (
	HASH = "e4be9e4db876e3e3179778b03e906297be5c8dbe"
	NAME = "ubuntu-18.04-desktop-amd64.iso"
	SIZE = 1921843200
)

func TestSqlite3Database_AddNewTorrent(t *testing.T) {
	infoHash, err := hex.DecodeString(HASH)
	checkErr(err, t)
	var files []File
	file := File{
		Path: NAME,
		Size: SIZE,
	}
	files = append(files, file)

	tearDown, db := setupTest(t)
	defer tearDown(t)

	err = db.AddNewTorrent(infoHash, NAME, files)
	checkErr(err, t)

	exist, err := db.DoesTorrentExist(infoHash)
	checkErr(err, t)
	if !exist {
		t.Fatal("expected torrent to exist")
	}

	n, err := db.GetNumberOfTorrents()
	checkErr(err, t)
	if n != 1 {
		t.Fatal("expected there to be 1 torrent")
	}

	metadata, err := db.GetTorrent(infoHash)
	checkErr(err, t)

	if metadata.Size != SIZE {
		t.Fatalf("Size mismatch. Expected: %d, Got: %d", SIZE, metadata.Size)
	}

	if metadata.Name != NAME {
		t.Fatalf("name mismatch. Expected: 1, Got: %s", metadata.Name)
	}

	if metadata.NFiles != 1 {
		t.Fatalf("filenumber mismatch. Expected: 1, Got: %d", metadata.NFiles)
	}
	now := time.Now().Unix()

	if metadata.DiscoveredOn > now {
		t.Fatalf("Timestamp is in the future. Now: %v, Got: %v", now, metadata.DiscoveredOn)
	}

	tFiles, err := db.GetFiles(infoHash)
	checkErr(err, t)

	if len(tFiles) != 1 {
		t.Fatalf("filenumber mismatch. Expected: 1, Got: %d", len(tFiles))
	}

	if tFiles[0].Size != SIZE {
		t.Fatalf("Size mismatch. Expected: %d, Got: %d", SIZE, tFiles[0].Size)
	}

	if tFiles[0].Path != NAME {
		t.Fatalf("name mismatch. Expected: 1, Got: %s", tFiles[0].Path)
	}
}

func TestSqlite3Database_DoesTorrentExist(t *testing.T) {
	tearDown, db := setupTest(t)
	defer tearDown(t)
	infoHash, err := hex.DecodeString("e4be9e4db876e3e3179778b03e906297be5c8dbe")
	exist, err := db.DoesTorrentExist(infoHash)
	checkErr(err, t)
	if exist {
		t.Fatal("torrent should not exist")
	}
}

func setupTest(t *testing.T) (func(t *testing.T), Database) {
	t.Log("setup db")
	loggerLevel := zap.NewAtomicLevel()
	// Logging levels: ("debug", "info", "warn", "error", "dpanic", "panic", and "fatal").
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(os.Stderr),
		loggerLevel,
	))
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	dbUrl, err := url.Parse("sqlite3:///tmp/test.db")
	checkErr(err, t)

	db, err := MakeDatabase(dbUrl, logger)
	checkErr(err, t)

	return func(t * testing.T) {
		os.Remove(dbUrl.Path)
		t.Log("teardown db")
	}, db
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}