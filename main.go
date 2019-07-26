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
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type cliOptions struct {
	app                 *kingpin.Application
	Config              string
	DefaultsFile        string
	Host                string
	Password            string
	ReadSamples         string
	SaveSamples         string
	User                string
	Databases           []string
	Port                int
	Seconds             int
	AllDatabases        bool
	AskPass             bool
	DisableSSL          bool
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

	connStr := buildConnString(opts)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Cannot connect to the database: %s\n", err)
		opts.app.Usage(os.Args[1:])
		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		fmt.Printf("Cannot connect to the database: %s\n", err)
		opts.app.Usage(os.Args[1:])
		os.Exit(1)
	}

	info, err := pginfo.New(db, opts.Databases, opts.Seconds)
	if err != nil {
		log.Fatal(err)
	}
	if opts.Verbose {
		info.SetLogLevel(logrus.InfoLevel)
	}
	if opts.Debug {
		info.SetLogLevel(logrus.DebugLevel)
	}

	errs := info.CollectGlobalInfo(db)
	if len(errs) > 0 {
		log.Println("Cannot collect info")
		for _, err := range errs {
			log.Println(err)
		}
	}

	for _, dbName := range info.DatabaseNames() {
		dbConnStr := fmt.Sprintf("%s database=%s", connStr, dbName)
		conn, err := sql.Open("postgres", dbConnStr)
		if err != nil {
			log.Errorf("Cannot connect to the %s database: %s", dbName, err)
			continue
		}
		if err := db.Ping(); err != nil {
			log.Errorf("Cannot connect to the %s database: %s", dbName, err)
			continue
		}
		if err := info.CollectPerDatabaseInfo(conn, dbName); err != nil {
			log.Errorf("Cannot collect information for the %s database: %s", dbName, err)
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

func buildConnString(opts cliOptions) string {
	parts := []string{}
	if opts.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", opts.User))
	}
	if opts.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", opts.Password))
	}

	if opts.DisableSSL {
		parts = append(parts, "sslmode=disable")
	}
	parts = append(parts, "dbname=postgres")

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
		StringVar(&opts.Host)
	app.Flag("list-encrypted-tables", "Include a list of the encrypted tables in all databases").
		Hidden().BoolVar(&opts.ListEncryptedTables)
	app.Flag("password", "Password to use when connecting").
		Short('W').
		StringVar(&opts.Password)
	app.Flag("port", "Port number to use for connection").
		Short('p').
		Default("5432").
		IntVar(&opts.Port)
	app.Flag("read-samples", "Create a report from the files found in this directory").
		Hidden().StringVar(&opts.ReadSamples) // hidden because it is not implemented yet
	app.Flag("save-samples", "Save the data files used to generate the summary in this directory").
		Hidden().StringVar(&opts.SaveSamples) // hidden because it is not implemented yet
	app.Flag("sleep", "Seconds to sleep when gathering status counters").
		Default("10").IntVar(&opts.Seconds)
	app.Flag("username", "User for login if not current user").
		Short('U').
		StringVar(&opts.User)
	app.Flag("disable-ssl", "Diable SSL for the connection").
		Default("true").BoolVar(&opts.DisableSSL)
	app.Flag("verbose", "Show verbose log").
		Default("false").BoolVar(&opts.Verbose)
	app.Flag("debug", "Show debug information in the logs").
		Default("false").BoolVar(&opts.Debug)
	_, err := app.Parse(args)
	return opts, err
}
