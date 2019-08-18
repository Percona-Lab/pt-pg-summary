package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/Percona-Lab/pt-pg-summary/internal/pginfo"
	"github.com/Percona-Lab/pt-pg-summary/templates"
	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type connOpts struct {
	Host       string
	Port       int
	User       string
	Password   string
	DisableSSL bool
}
type cliOptions struct {
	app                 *kingpin.Application
	connOpts            connOpts
	Config              string
	DefaultsFile        string
	ReadSamples         string
	SaveSamples         string
	Databases           []string
	Seconds             int
	AllDatabases        bool
	AskPass             bool
	ListEncryptedTables bool
	Verbose             bool
	Debug               bool
}

func main() {
	opts, err := parseCommandLineOpts(os.Args[1:])
	if err != nil {
		fmt.Printf("Cannot parse command line arguments: %s", err)
		os.Exit(1)
	}
	logger := logrus.New()
	if opts.Verbose {
		logger.SetLevel(logrus.InfoLevel)
	}
	if opts.Debug {
		logger.SetLevel(logrus.DebugLevel)
	}

	dsn := buildConnString(opts.connOpts, "postgres")
	logger.Infof("Connecting to the database server using: %s", safeConnString(opts.connOpts, "postgres"))

	db, err := connect(dsn)
	if err != nil {
		logger.Errorf("Cannot connect to the database: %s\n", err)
		opts.app.Usage(os.Args[1:])
		os.Exit(1)
	}
	logger.Infof("Connection OK")

	info, err := pginfo.NewWithLogger(db, opts.Databases, opts.Seconds, logger)
	if err != nil {
		log.Fatalf("Cannot create a data collector instance: %s", err)
	}

	logger.Info("Getting global information")
	errs := info.CollectGlobalInfo(db)
	if len(errs) > 0 {
		logger.Errorf("Cannot collect info")
		for _, err := range errs {
			logger.Error(err)
		}
	}

	logger.Info("Collecting per database information")
	logger.Debugf("Will collect information for these databases: (%T), %v", info.DatabaseNames(), info.DatabaseNames())
	for _, dbName := range info.DatabaseNames() {
		dsn := buildConnString(opts.connOpts, dbName)
		logger.Infof("Connecting to the %q database", dbName)
		conn, err := connect(dsn)
		if err != nil {
			logger.Errorf("Cannot connect to the %s database: %s", dbName, err)
			continue
		}
		if err := info.CollectPerDatabaseInfo(conn, dbName); err != nil {
			logger.Errorf("Cannot collect information for the %s database: %s", dbName, err)
		}
		conn.Close()
	}

	masterTmpl, err := template.New("master").Funcs(funcsMap()).Parse(templates.TPL)
	if err != nil {
		log.Fatal(err)
	}

	if err := masterTmpl.ExecuteTemplate(os.Stdout, "report", info); err != nil {
		log.Fatal(err)
	}

}

func connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect to the database")
	}

	if err := db.Ping(); err != nil {
		return nil, errors.Wrap(err, "cannot connect to the database")
	}
	return db, nil
}

func funcsMap() template.FuncMap {
	return template.FuncMap{
		"trim": func(s string, size int) string {
			if len(s) < size {
				return s
			}
			return s[:size]
		},
	}
}

func buildConnString(opts connOpts, dbName string) string {
	parts := []string{}
	if opts.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", opts.Host))
	}
	if opts.Port != 0 {
		parts = append(parts, fmt.Sprintf("port=%d", opts.Port))
	}
	if opts.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", opts.User))
	}
	if opts.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", opts.Password))
	}
	if opts.DisableSSL {
		parts = append(parts, "sslmode=disable")
	}
	if dbName == "" {
		dbName = "postgres"
	}
	parts = append(parts, fmt.Sprintf("dbname=%s", dbName))

	return strings.Join(parts, " ")
}

// build the same connection string as buildConnString but the password is hidden so
// we can display this in the logs
func safeConnString(opts connOpts, dbName string) string {
	parts := []string{}
	if opts.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", opts.Host))
	}
	if opts.Port != 0 {
		parts = append(parts, fmt.Sprintf("port=%d", opts.Port))
	}
	if opts.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", opts.User))
	}
	if opts.Password != "" {
		parts = append(parts, "password=******")
	}
	if opts.DisableSSL {
		parts = append(parts, "sslmode=disable")
	}
	if dbName == "" {
		dbName = "postgres"
	}
	parts = append(parts, fmt.Sprintf("dbname=%s", dbName))

	return strings.Join(parts, " ")
}

func parseCommandLineOpts(args []string) (cliOptions, error) {
	app := kingpin.New("pt-pg-summary", "Percona Toolkit - PostgreSQL Summary")
	// version, commit and date will be set at build time by the compiler -ldflags param
	app.Version(fmt.Sprintf("%s version %s, git commit %s, date: %s", app.Name, version, commit, date))
	opts := cliOptions{app: app}

	app.Flag("ask-pass", "Prompt for a password when connecting to PostgreSQL").
		Hidden().BoolVar(&opts.AskPass) // hidden because it is not implemented yet
	app.Flag("config", "Config file").
		Hidden().StringVar(&opts.Config) // hidden because it is not implemented yet
	app.Flag("databases", "Summarize this comma-separated list of databases. All if not specified").
		StringsVar(&opts.Databases)
	app.Flag("defaults-file", "Only read PostgreSQL options from the given file").
		Hidden().StringVar(&opts.DefaultsFile) // hidden because it is not implemented yet
	app.Flag("host", "Host to connect to").
		Short('h').
		StringVar(&opts.connOpts.Host)
	app.Flag("list-encrypted-tables", "Include a list of the encrypted tables in all databases").
		Hidden().BoolVar(&opts.ListEncryptedTables)
	app.Flag("password", "Password to use when connecting").
		Short('W').
		StringVar(&opts.connOpts.Password)
	app.Flag("port", "Port number to use for connection").
		Short('p').
		Default("5432").
		IntVar(&opts.connOpts.Port)
	app.Flag("read-samples", "Create a report from the files found in this directory").
		Hidden().StringVar(&opts.ReadSamples) // hidden because it is not implemented yet
	app.Flag("save-samples", "Save the data files used to generate the summary in this directory").
		Hidden().StringVar(&opts.SaveSamples) // hidden because it is not implemented yet
	app.Flag("sleep", "Seconds to sleep when gathering status counters").
		Default("10").IntVar(&opts.Seconds)
	app.Flag("username", "User for login if not current user").
		Short('U').
		StringVar(&opts.connOpts.User)
	app.Flag("disable-ssl", "Diable SSL for the connection").
		Default("true").BoolVar(&opts.connOpts.DisableSSL)
	app.Flag("verbose", "Show verbose log").
		Default("false").BoolVar(&opts.Verbose)
	app.Flag("debug", "Show debug information in the logs").
		Default("false").BoolVar(&opts.Debug)
	_, err := app.Parse(args)

	dbs := []string{}
	for _, databases := range opts.Databases {
		ds := strings.Split(databases, ",")
		dbs = append(dbs, ds...)
	}
	opts.Databases = dbs
	return opts, err
}
