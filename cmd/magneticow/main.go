package main

import (
	"encoding/hex"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Wessie/appdirs"
	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/izolight/magnetico/pkg/persistence"
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
	BindAddr    string
	Verbosity   int
}

// ========= TD: TemplateData =========
type HomepageTD struct {
	Count uint
}

type TorrentsTD struct {
	Search            string
	SubscriptionURL   string
	Torrents          []persistence.TorrentMetadata
	Epoch             int64
	OrderBy           string
	Ascending         bool
	Limit             uint
	FirstOrderedValue interface{}
	LastOrderedValue  interface{}
	StartID           uint64
	FirstID           uint64
	LastID            uint64
	IsFirstPage       bool
	NextPageExists    bool
}

type TorrentTD struct {
	Torrent persistence.TorrentMetadata
	Files   []persistence.File
}

type FeedTD struct {
}

type StatisticsTD struct {
	Stats persistence.Statistics
	Dates []string
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
	router.HandleFunc("/feed", feedHandler)
	router.PathPrefix("/static").HandlerFunc(staticHandler)

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
			return tm.Format("2006-01-02 15:04")
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

	http.ListenAndServe(opFlags.BindAddr, handlers.CombinedLoggingHandler(os.Stdout, router))
}

// DONE
func rootHandler(w http.ResponseWriter, r *http.Request) {
	count, err := database.GetNumberOfTorrents()
	if err != nil {
		zap.L().Error("Couldn't get number of torrents",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
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
	backward := false
	limit := uint(N_TORRENTS)

	var lastOrderedValue uint64
	var startID, firstID, lastID uint64

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
	if queryValues.Get("backward") != "" {
		backward = true
	}

	qLimit := queryValues.Get("limit")
	if qLimit != "" {
		l, err := strconv.ParseUint(qLimit, 10, 64)
		if err != nil {
			zap.L().Error("Couldn't parse limit",
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		limit = uint(l)
	}

	if queryValues.Get("epoch") != "" {
		qEpoch, err := strconv.ParseInt(queryValues.Get("epoch"), 10, 64)
		if err != nil {
			zap.L().Error("Couldn't parse epoch",
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		epoch = time.Unix(qEpoch, 0)

		qLastOrderedValue := queryValues.Get("lastOrderedValue")
		qLastID := queryValues.Get("lastID")

		if qLastOrderedValue != "" && qLastID != "" {
			lastOrderedValue, err = strconv.ParseUint(qLastOrderedValue, 10, 64)
			if err != nil {
				zap.L().Error("Couldn't parse lastOrderedValue",
					zap.Error(err),
				)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			lastID, err = strconv.ParseUint(qLastID, 10, 64)
			if err != nil {
				zap.L().Error("Couldn't parse lastID",
					zap.Error(err),
				)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("when specifying epoch, need to supply lastOrderedValue and lastID as well"))
		}
	}

	var torrents []persistence.TorrentMetadata
	torrents, err = database.QueryTorrents(
		search,
		epoch.Unix(),
		orderBy,
		ascending,
		limit,
		lastOrderedValue,
		lastID,
		backward,
	)
	if err != nil {
		zap.L().Error("Couldn't get torrents from database",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if queryValues.Get("startID") != "" {
		startID, err = strconv.ParseUint(queryValues.Get("startID"), 10, 64)
	} else if len(torrents)!= 0 {
		startID = uint64(torrents[0].ID)
	}

	var firstOrdered, lastOrdered interface{}
	var isFirstPage bool
	if len(torrents) != 0 {
		firstOrdered = torrents[0].DiscoveredOn
		lastOrdered = torrents[len(torrents)-1].DiscoveredOn
		firstID = uint64(torrents[0].ID)
		lastID = uint64(torrents[len(torrents)-1].ID)
		isFirstPage = startID == uint64(torrents[0].ID) || startID == uint64(torrents[len(torrents)-1].ID)
	}

	// TODO: for testing, REMOVE
	//torrents[2].HasReadme = true

	templates["torrents"].Execute(w, TorrentsTD{
		Search:            search,
		SubscriptionURL:   "borabora",
		Torrents:          torrents,
		Epoch:             epoch.Unix(),
		OrderBy:           qOrderBy,
		Ascending:         ascending,
		Limit:             limit,
		FirstOrderedValue: firstOrdered,
		LastOrderedValue:  lastOrdered,
		StartID:           startID,
		FirstID:           firstID,
		LastID:            lastID,
		NextPageExists:    len(torrents) >= 20,
		IsFirstPage:       isFirstPage,
	})

}

func torrentsInfohashHandler(w http.ResponseWriter, r *http.Request) {
	// show torrents/{infohash}
	infoHash, err := hex.DecodeString(mux.Vars(r)["infohash"])
	if err != nil {
		zap.L().Error("Couldn't decode infohash",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	torrent, err := database.GetTorrent(infoHash)
	if err != nil {
		zap.L().Error("Couldn't get torrent from database",
			zap.Error(err),
			zap.String("infohash", string(infoHash)),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if torrent == nil {
		http.NotFound(w, r)
		return
	}

	files, err := database.GetFiles(infoHash)
	if err != nil {
		zap.L().Error("Couldn't get files from database",
			zap.Error(err),
			zap.String("infohash", string(infoHash)),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	templates["torrent"].Execute(w, TorrentTD{
		Torrent: *torrent,
		Files:   files,
	})
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {
	interval, err := time.ParseDuration("24h")
	if err != nil {
		zap.L().Error("Couldn't parse Interval",
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	from := time.Now().Add(-30 * interval)

	stats, err := database.GetStatistics(30, from.Format("2006-01-02"))
	if err != nil {
		zap.L().Error("Couldn't get parse dateformat",
			zap.Error(err),
			zap.Time("date", from),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var dates []string
	for i := 0; i < 30; i++ {
		from = from.Add(interval)
		dates = append(dates, from.Format("2006-01-02"))
	}
	templates["statistics"].Execute(w, StatisticsTD{
		Stats: *stats,
		Dates: dates,
	})
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
	w.Header().Set("Cache-Control", "max-age=86400")
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
