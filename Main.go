package main

import (
	"database/sql"
	_ "image/jpeg"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/juju/ratelimit"
	"gopkg.in/mgutz/dat.v2/dat"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

var calls int
var ratelimits map[string]*ratelimit.Bucket
var iplimits map[string]*ratelimit.Bucket

var dbpswd = "ftT6A4MrF6hPt"

// global database (pooling provided by SQL driver)
var database *runner.DB

var node *snowflake.Node

func init() {
	// create a normal database connection through database/sql
	db, err := sql.Open("postgres", "dbname=aidex user=admin password="+dbpswd+" host=admin.rwbyadventures.com")
	if err != nil {
		panic(err)
	}
	node, _ = snowflake.NewNode(5)

	// ensures the database can be pinged with an exponential backoff (15 min)
	runner.MustPing(db)

	// set to reasonable values for production
	db.SetMaxIdleConns(4)
	db.SetMaxOpenConns(50)

	// set this to enable interpolation
	dat.EnableInterpolation = true

	// set to check things like sessions closing.
	// Should be disabled in production/release builds.
	dat.Strict = false

	// Log any query over 10ms as warnings. (optional)
	runner.LogQueriesThreshold = 500 * time.Millisecond

	database = runner.NewDB(db, "postgres")
}

func main() {
	go hostService()
	ratelimits = make(map[string]*ratelimit.Bucket)
	iplimits = make(map[string]*ratelimit.Bucket)
	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}
