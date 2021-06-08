package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "image/jpeg"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/juju/ratelimit"
	"github.com/plutov/paypal/v4"
	"github.com/pmylund/go-cache"
	ipn "github.com/webhookrelay/paypal-ipn"
	"gopkg.in/mgutz/dat.v2/dat"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

var calls int
var ratelimits map[string]*ratelimit.Bucket
var iplimits map[string]*ratelimit.Bucket

var dbpswd = "ftT6A4MrF6hPt"

var canProcessPaypal = true

var paypalClient *paypal.Client

// global database (pooling provided by SQL driver)
var database *runner.DB

var node *snowflake.Node

var urlcache *cache.Cache

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

	urlcache = cache.New(5*time.Minute, 10*time.Minute)

	ratelimits = make(map[string]*ratelimit.Bucket)
	iplimits = make(map[string]*ratelimit.Bucket)

	//Connect to paypal
	var err error
	paypalClient, err = paypal.NewClient(paypalClientID, paypalClientSecret, paypal.APIBaseLive)
	if err != nil {
		fmt.Println("Error connecting to Paypal")
		canProcessPaypal = false
	}
	paypalClient.SetLog(os.Stdout) // Set log to terminal stdout
	paypalClient.GetAccessToken(context.Background())

	mux := http.NewServeMux()
	srv := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	listener := ipn.New(false)

	mux.Handle("/", listener.WebhooksHandler(paypalWebhook))
	log.Println("server starting on :8080")

	go srv.ListenAndServe()
	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}
