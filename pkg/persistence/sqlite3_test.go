package persistence

import (
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	HASH = "e4be9e4db876e3e3179778b03e906297be5c8dbe"
	NAME = "ubuntu-18.04-desktop-amd64.iso"
	SIZE = 1921843200
)

func TestSqlite3Database_AddNewTorrent(t *testing.T) {
	tearDown, db := setupTest(t)
	defer tearDown(t)

	addTorrent(db, t)
}

func TestSqlite3Database_DoesTorrentExist(t *testing.T) {
	infoHash, err := hex.DecodeString(HASH)

	tearDown, db := setupTest(t)
	defer tearDown(t)

	exist, err := db.DoesTorrentExist(infoHash)
	checkErr(err, t)
	if exist {
		t.Fatal("expected torrent to not exist")
	}

	addTorrent(db, t)
	exist, err = db.DoesTorrentExist(infoHash)
	checkErr(err, t)
	if !exist {
		t.Fatal("expected torrent to exist")
	}
}

func TestSqlite3Database_GetNumberOfTorrents(t *testing.T) {
	tearDown, db := setupTest(t)
	defer tearDown(t)

	addTorrent(db, t)

	n, err := db.GetNumberOfTorrents()
	checkErr(err, t)
	if n != 1 {
		t.Fatal("expected there to be 1 torrent")
	}
}

func TestSqlite3Database_GetTorrent(t *testing.T) {
	infoHash, err := hex.DecodeString(HASH)

	tearDown, db := setupTest(t)
	defer tearDown(t)

	addTorrent(db, t)

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
}

func TestSqlite3Database_GetFiles(t *testing.T) {
	infoHash, err := hex.DecodeString(HASH)

	tearDown, db := setupTest(t)
	defer tearDown(t)

	addTorrent(db, t)

	tFiles, err := db.GetFiles(infoHash)
	checkErr(err, t)

	t.Log(tFiles)

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

func TestSqlite3Database_GetStatistics(t *testing.T) {
	tearDown, db := setupTest(t)
	defer tearDown(t)

	addTorrent(db, t)

	dataPoints := uint(2)

	// start is from 31.12.2016
	stats, err := db.GetStatistics(dataPoints, "2016")
	checkErr(err, t)

	if stats.N != dataPoints {
		t.Fatalf("expected there to be %d datapoints, got %d", dataPoints, stats.N)
	}

	if stats.NFiles[0] != 0 && stats.TotalSize[0] != 0 && stats.NTorrents[0] != 0 {
		t.Fatalf("expected there to be no torrents in previous year")
	}

	if stats.NFiles[1] != 1 && stats.TotalSize[1] != SIZE && stats.NTorrents[1] != 1 {
		t.Fatalf("expected there to be our torrent")
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
	// ensure file does not exist before
	os.Remove(dbUrl.Path)

	db, err := MakeDatabase(dbUrl, logger)
	checkErr(err, t)

	return func(t *testing.T) {
		os.Remove(dbUrl.Path)
		t.Log("teardown db")
	}, db
}

func addTorrent(db Database, t *testing.T) {
	infoHash, err := hex.DecodeString(HASH)
	checkErr(err, t)
	var files []File
	file := File{
		Path: NAME,
		Size: SIZE,
	}
	files = append(files, file)

	err = db.AddNewTorrent(infoHash, NAME, files)
	checkErr(err, t)
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}
