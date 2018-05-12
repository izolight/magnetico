package main

import (
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Wessie/appdirs"
	"github.com/boramalper/magnetico/pkg/persistence"
	"github.com/jessevdk/go-flags"
	"net/url"
	"path"
	"net"
)

const N_TORRENTS = 20

var templates map[string]*template.Template
var database persistence.Database

type cmdFlags struct {
	DatabaseURL string `long:"database" description:"URL of the database."`
	BindAddr    string `short:"b" long:"bind" description:"Address that the WebUI should listen on." env:"BIND_ADDR" env-delim:"," default:"0.0.0.0:8080"`
	Verbose     []bool `short:"v" long:"verbose" description:"Increases verbosity."`
}

type opFlags struct {
	DatabaseURL *url.URL
	BindAddr	string
	Verbosity   int
}

// ========= TD: TemplateData =========
type HomepageTD struct {
	Count uint
}

type TorrentsTD struct {
	Search           string
	SubscriptionURL  string
	Torrents         []persistence.TorrentMetadata
	Epoch            int64
	OrderBy          string
	Ascending        bool
	Limit            uint
	PreviousOrderedValue interface{}
	LastOrderedValue interface{}
	FirstID			 uint64
	PreviousID		 uint64
	LastID           uint64
	IsFirstPage		 bool
	NextPageExists   bool
}

type TorrentTD struct {
}

type FeedTD struct {
}

type StatisticsTD struct {
}

func main() {
	loggerLevel := zap.NewAtomicLevel()
	// Logging levels: ("debug", "info", "warn", "error", "dpanic", "panic", and "fatal").
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(os.Stderr),
		loggerLevel,
	))
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	opFlags, err := parseFlags()
	if err != nil {
		return
	}

	zap.L().Info("magneticow v0.7.0 has been started.")
	zap.L().Info("Copyright (C) 2017  Mert Bora ALPER <bora@boramalper.org>.")
	zap.L().Info("Dedicated to Cemile Binay, in whose hands I thrived.")

	switch opFlags.Verbosity {
	case 0:
		loggerLevel.SetLevel(zap.WarnLevel)
	case 1:
		loggerLevel.SetLevel(zap.InfoLevel)
	default: // Default: i.e. in case of 2 or more.
		// TODO: print the caller (function)'s name and line number!
		loggerLevel.SetLevel(zap.DebugLevel)
	}

	zap.ReplaceGlobals(logger)

	router := mux.NewRouter()
	router.HandleFunc("/", rootHandler)
	router.HandleFunc("/torrents", torrentsHandler)
	router.HandleFunc("/torrents/{infohash:[a-z0-9]{40}}", torrentsInfohashHandler)
	router.HandleFunc("/statistics", statisticsHandler)

	router.PathPrefix("/static").HandlerFunc(staticHandler)

	router.HandleFunc("/feed", feedHandler)

	templateFunctions := template.FuncMap{
		"add": func(augend int, addends int) int {
			return augend + addends
		},

		"subtract": func(minuend int, subtrahend int) int {
			return minuend - subtrahend
		},

		"bytesToHex": func(bytes []byte) string {
			return hex.EncodeToString(bytes)
		},

		"unixTimeToYearMonthDay": func(s int64) string {
			tm := time.Unix(s, 0)
			// > Format and Parse use example-based layouts. Usually you’ll use a constant from time
			// > for these layouts, but you can also supply custom layouts. Layouts must use the
			// > reference time Mon Jan 2 15:04:05 MST 2006 to show the pattern with which to
			// > format/parse a given time/string. The example time must be exactly as shown: the
			// > year 2006, 15 for the hour, Monday for the day of the week, etc.
			// https://gobyexample.com/time-formatting-parsing
			// Why you gotta be so weird Go?
			return tm.Format("02/01/2006")
		},

		"humanizeSize": func(s uint64) string {
			return humanize.IBytes(s)
		},
	}

	templates = make(map[string]*template.Template)
	templates["feed"] = template.Must(template.New("feed").Parse(string(mustAsset("templates/feed.xml"))))
	templates["homepage"] = template.Must(template.New("homepage").Parse(string(mustAsset("templates/homepage.html"))))
	templates["statistics"] = template.Must(template.New("statistics").Parse(string(mustAsset("templates/statistics.html"))))
	templates["torrent"] = template.Must(template.New("torrent").Funcs(templateFunctions).Parse(string(mustAsset("templates/torrent.html"))))
	templates["torrents"] = template.Must(template.New("torrents").Funcs(templateFunctions).Parse(string(mustAsset("templates/torrents.html"))))

	database, err = persistence.MakeDatabase(opFlags.DatabaseURL, logger)
	if err != nil {
		panic(err.Error())
	}

	zap.L().Info("magneticow is ready to serve!")
	http.ListenAndServe(opFlags.BindAddr, router)
}

// DONE
func rootHandler(w http.ResponseWriter, r *http.Request) {
	count, err := database.GetNumberOfTorrents()
	if err != nil {
		panic(err.Error())
	}
	templates["homepage"].Execute(w, HomepageTD{
		Count: count,
	})
}

func torrentsHandler(w http.ResponseWriter, r *http.Request) {
	queryValues := r.URL.Query()

	search := queryValues.Get("search")
	epoch := time.Now()
	orderBy := persistence.ByRelevance
	ascending := false
	limit := uint(20)

	var previousOrderedValue, lastOrderedValue uint64
	var firstID, previousID, lastID uint64

	var err error

	qOrderBy := queryValues.Get("orderBy")
	switch qOrderBy {
	case "size":
		orderBy = persistence.BySize
	case "discovered":
		orderBy = persistence.ByDiscoveredOn
	case "files":
		orderBy = persistence.ByNFiles
	case "seeders":
		orderBy = persistence.ByNSeeders
	case "leechers":
		orderBy = persistence.ByNLeechers
	default:
		if search == "" {
			orderBy = persistence.ByDiscoveredOn
		}
	}

	if queryValues.Get("ascending") != "" {
		ascending = true
	}

	qLimit := queryValues.Get("limit")
	if qLimit != "" {
		l, err := strconv.ParseUint(qLimit, 10, 64)
		if err != nil {
			panic(err.Error())
		}
		limit = uint(l)
	}

	forward := true

	if queryValues.Get("epoch") != "" {
		qEpoch, err := strconv.ParseInt(queryValues.Get("epoch"), 10, 64)
		if err != nil {
			panic(err.Error())
		}
		epoch = time.Unix(qEpoch, 0)

		qLastOrderedValue := queryValues.Get("lastOrderedValue")
		qLastID := queryValues.Get("lastID")
		qPreviousOrderedValue := queryValues.Get("previousOrderedValue")
		qPreviousID := queryValues.Get("previousID")

		if qLastOrderedValue != "" && qLastID != "" {
			lastOrderedValue, err = strconv.ParseUint(qLastOrderedValue, 10, 64)
			if err != nil {
				panic(err.Error())
			}
			lastID, err = strconv.ParseUint(qLastID, 10, 64)
			if err != nil {
				panic(err.Error())
			}
		} else if qPreviousOrderedValue != "" && qPreviousID != "" {
			previousOrderedValue, err = strconv.ParseUint(qPreviousOrderedValue, 10, 64)
			if err != nil {
				panic(err.Error())
			}
			previousID, err = strconv.ParseUint(qPreviousID, 10, 64)
			if err != nil {
				panic(err.Error())
			}
			forward = false
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("when specifying epoch, need to supply lastOrderedValue and lastID as well"))
		}
	}

	var torrents []persistence.TorrentMetadata
	if forward {
		torrents, err = database.QueryTorrents(
			search,
			epoch.Unix(),
			orderBy,
			ascending,
			limit,
			lastOrderedValue,
			lastID,
		)
		if err != nil {
			panic(err.Error())
		}
	} else {
		torrents, err = database.QueryTorrents(
			search,
			epoch.Unix(),
			orderBy,
			ascending,
			limit,
			previousOrderedValue,
			previousID,
		)
		if err != nil {
			panic(err.Error())
		}
	}

	if queryValues.Get("firstID") != "" {
		firstID, err = strconv.ParseUint(queryValues.Get("firstID"),10, 64)
	} else {
		firstID = uint64(torrents[0].ID)
	}

	// TODO: for testing, REMOVE
	//torrents[2].HasReadme = true

	templates["torrents"].Execute(w, TorrentsTD{
		Search:           search,
		SubscriptionURL:  "borabora",
		Torrents:         torrents,
		Epoch:            epoch.Unix(),
		OrderBy:          qOrderBy,
		Ascending:        ascending,
		Limit:            limit,
		PreviousOrderedValue: torrents[0].DiscoveredOn,
		LastOrderedValue: torrents[len(torrents)-1].DiscoveredOn,
		FirstID:          firstID,
		PreviousID:       uint64(torrents[0].ID),
		LastID:           uint64(torrents[len(torrents)-1].ID),
		NextPageExists:   len(torrents) >= 20,
		IsFirstPage: 	  firstID == uint64(torrents[0].ID),
	})

}

func torrentsInfohashHandler(w http.ResponseWriter, r *http.Request) {
	// show torrents/{infohash}
	infoHash, err := hex.DecodeString(mux.Vars(r)["infohash"])
	if err != nil {
		panic(err.Error())
	}

	torrent, err := database.GetTorrent(infoHash)
	if err != nil {
		panic(err.Error())
	}

	templates["torrent"].Execute(w, torrent)
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {

}

func feedHandler(w http.ResponseWriter, r *http.Request) {

}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Asset(r.URL.Path[1:])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var contentType string
	if strings.HasSuffix(r.URL.Path, ".css") {
		contentType = "text/css; charset=utf-8"
	} else { // fallback option
		contentType = http.DetectContentType(data)
	}
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

func mustAsset(name string) []byte {
	data, err := Asset(name)
	if err != nil {
		log.Panicf("Could NOT access the requested resource `%s`: %s (please inform us, this is a BUG!)", name, err.Error())
	}
	return data
}

func parseFlags() (*opFlags, error) {
	opF := new(opFlags)
	cmdF := new(cmdFlags)

	_, err := flags.Parse(cmdF)
	if err != nil {
		return nil, err
	}

	if cmdF.DatabaseURL == "" {
		cmdF.DatabaseURL = "sqlite3://" + path.Join(
			appdirs.UserDataDir("magneticod", "", "", false),
			"database.sqlite3",
		)
	}
	opF.DatabaseURL, err = url.Parse(cmdF.DatabaseURL)
	if err != nil {
		zap.L().Fatal("Failed to parse DB URL", zap.Error(err))
	}

	_, err = net.ResolveTCPAddr("tcp", cmdF.BindAddr)
	if err != nil {
		zap.L().Fatal("Failed to parse Address", zap.Error(err))
	}
	opF.BindAddr = cmdF.BindAddr

	opF.Verbosity = len(cmdF.Verbose)

	return opF, nil
}
