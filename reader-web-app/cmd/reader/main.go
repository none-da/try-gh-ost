package main

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/none-da/try-online-schema-change/reader-web-app/pkg/reader"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

// StartTime gives the start time of server
var StartTime = time.Now()

const defaultAppPort string = "8080"

func validateEnvVars() map[string]string {
	var requiredDbEnvVars = [5]string{
		"DB_NAME",
		"DB_HOST",
		"DB_USER",
		"DB_PASSWORD",
		"DB_PORT"}

	dbConfig := make(map[string]string)
	for _, envVar := range requiredDbEnvVars {
		envVarValue, isSet := os.LookupEnv(envVar)
		if !isSet {
			log.Fatal(fmt.Sprintf("%s is a required env variable. Please set it and with a valid value ofcourse.", envVar))
		}
		dbConfig[envVar] = envVarValue
	}

	return dbConfig
}

func uptime() string {
	elapsedTime := time.Since(StartTime)
	return fmt.Sprintf("%d:%d:%d", int(math.Round(elapsedTime.Hours())), int(math.Round(elapsedTime.Minutes())), int(math.Round(elapsedTime.Seconds())))
}

func homePage(w http.ResponseWriter, req *http.Request) {
	host, _ := os.Hostname()
	fmt.Fprintf(w, fmt.Sprintf("[HOST: %s] (uptime: %s)]", host, uptime()))
}

func (env *Env) readData(w http.ResponseWriter, r *http.Request) {
	baseItem := reader.ReadData(env.db)
	fmt.Fprintf(w, fmt.Sprintf("created_on: %s updated_on: %s product_id: %s", baseItem.CreatedOn, baseItem.UpdatedOn, baseItem.ProductID))
}

// Create a custom Env struct which holds a connection pool.
type Env struct {
	db *sql.DB
}

func main() {
	var requiredLogLevel string
	flag.StringVarP(&requiredLogLevel, "log-level", "l", "info", "Required log level. debug/info/warn/error.")
	var printHelp bool
	flag.BoolVarP(&printHelp, "help", "h", false, "Prints this help content.")
	maxOpenConnectionsToDbPtr := flag.Int("max-open-conns-db", 200, "Max Open Connections to the Db.")
	maxIdleConnectionsToDbPtr := flag.Int("max-idle-conns-db", 10, "Max Idle Connections to the Db.")
	var sleepTimeBetweenInsertsPtr int64
	flag.Int64VarP(&sleepTimeBetweenInsertsPtr, "sleep-between-inserts", "s", 0, "sleep time in seconds between inserts, to do db commits")
	var rowsToInsert int64
	flag.Int64VarP(&rowsToInsert, "rows-to-insert", "r", 1, "Rows to insert.")
	flag.Parse()
	if printHelp {
		flag.Usage()
		return
	}
	switch requiredLogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat:   time.RFC3339,
		DisableHTMLEscape: true,
	})
	log.SetOutput(os.Stdout)
	log.Info("I am a reader")

	dbConfig := validateEnvVars()

	dsnString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", dbConfig["DB_USER"], dbConfig["DB_PASSWORD"], dbConfig["DB_HOST"], dbConfig["DB_PORT"], dbConfig["DB_NAME"])
	log.Debug("dsnString: ", dsnString)
	db, err := sql.Open("mysql", dsnString)
	reader.CheckErr(err)
	db.SetMaxOpenConns(*maxOpenConnectionsToDbPtr)
	db.SetMaxIdleConns(*maxIdleConnectionsToDbPtr)

	err = db.Ping()
	reader.CheckErr(err)

	env := &Env{db: db}

	http.HandleFunc("/", homePage)
	http.HandleFunc("/read", env.readData)

	http.ListenAndServe(fmt.Sprintf(":%s", defaultAppPort), nil)
}
