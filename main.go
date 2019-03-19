package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Percona-Lab/pt-pg-summary/models"
	"github.com/Percona-Lab/pt-pg-summary/templates"
	"github.com/alecthomas/kingpin"
	version "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

type templateData struct {
	ClusterInfo        []*models.ClusterInfo
	ConnectedClients   []*models.ConnectedClients
	DatabaseWaitEvents []*models.DatabaseWaitEvents
	Databases          []*models.Databases
	GlobalWaitEvents   []*models.GlobalWaitEvents
	PortAndDatadir     *models.PortAndDatadir
	SlaveHosts96       []*models.SlaveHosts96
	SlaveHosts10       []*models.SlaveHosts10
	Tablespaces        []*models.Tablespaces
	Counters           map[models.Name][]*models.Counters    // Counters per database
	IndexCacheHitRatio map[string]*models.IndexCacheHitRatio // Indexes cache hit ratio per database
	TableCacheHitRatio map[string]*models.TableCacheHitRatio // Tables cache hit ratio per database
	TableAccess        map[string][]*models.TableAccess      // Table access per database
	AllDatabases       bool
	ServerVersion      *version.Version
	Sleep              int

	// This is the list of databases from where we should get Table Cache Hit, Index Cache Hits, etc.
	// This field is being populated on the newData function depending on the cli parameters.
	// If --databases was not specified, this array will have the list of ALL databases from the GetDatabases
	// method in the models pkg
	databases []string
}

type cliOptions struct {
	Config              string
	DefaultsFile        string
	Host                string
	Password            string
	ReadSamples         string
	SaveSamples         string
	Socket              string
	User                string
	Databases           []string
	Port                int
	Seconds             int
	AllDatabases        bool
	AskPass             bool
	DisableSSL          bool
	ListEncryptedTables bool
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
		log.Fatal(err)
	}

	data, err := newData(db, opts.Databases, opts.Seconds)
	if err != nil {
		log.Fatal(err)
	}

	errs := collect(db, data)
	if len(errs) > 0 {
		log.Println("Cannot collect info")
		for _, err := range errs {
			log.Println(err)
		}
	}

	for _, dbName := range data.databases {
		dbConnStr := fmt.Sprintf("%s database=%s", connStr, dbName)
		conn, err := sql.Open("postgres", dbConnStr)
		if err != nil {
			log.Errorf("Cannot connect to the %s database: %s", dbName, err)
		}
		if err := collectPerDatabaseInfo(conn, dbName, data); err != nil {
			log.Errorf("Cannot collect information for the %s database: %s", dbName, err)
		}
		conn.Close()
	}

	masterTmpl, err := template.New("master").Funcs(funcsMap()).Parse(templates.TPL)
	if err != nil {
		log.Fatal(err)
	}

	if err := masterTmpl.ExecuteTemplate(os.Stdout, "report", data); err != nil {
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
	if opts.Socket == "" {
		if opts.Host == "" {
			opts.Host = "127.0.0.1"
		}
		parts = append(parts, fmt.Sprintf("host=%s", opts.Host))
		if opts.Port == 0 {
			parts = append(parts, fmt.Sprintf("port=%d", opts.Port))
		}
	} else {
		parts = append(parts, fmt.Sprintf("host=%s", opts.Socket))
	}

	if opts.DisableSSL {
		parts = append(parts, "sslmode=disable")
	}

	return strings.Join(parts, " ")
}

func newData(db models.XODB, databases []string, sleep int) (*templateData, error) {
	var err error
	data := &templateData{
		Counters:           make(map[models.Name][]*models.Counters),
		TableAccess:        make(map[string][]*models.TableAccess),
		TableCacheHitRatio: make(map[string]*models.TableCacheHitRatio),
		IndexCacheHitRatio: make(map[string]*models.IndexCacheHitRatio),
		Sleep:              sleep,
	}

	if data.Databases, err = models.GetDatabases(db); err != nil {
		return nil, errors.Wrap(err, "Cannot get databases list")
	}

	if len(databases) < 1 {
		data.databases = make([]string, 0, len(data.Databases))
		allDatabases, err := models.GetAllDatabases(db)
		if err != nil {
			return nil, errors.Wrap(err, "cannot get the list of all databases")
		}
		for _, database := range allDatabases {
			data.databases = append(data.databases, string(database.Datname))
		}
	} else {
		copy(data.databases, databases)
	}

	serverVersion, err := models.GetServerVersion(db)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot get the connected clients list")
	}

	if data.ServerVersion, err = parseServerVersion(serverVersion.Version); err != nil {
		return nil, fmt.Errorf("cannot get server version: %s", err.Error())
	}

	return data, nil
}

func collectPerDatabaseInfo(db models.XODB, dbName string, data *templateData) (err error) {
	if data.TableAccess[dbName], err = models.GetTableAccesses(db); err != nil {
		return errors.Wrapf(err, "cannot get Table Accesses for the %s database", dbName)
	}

	if data.TableCacheHitRatio[dbName], err = models.GetTableCacheHitRatio(db); err != nil {
		return errors.Wrapf(err, "cannot get Table Cache Hit Ratios for the %s database", dbName)
	}

	if data.IndexCacheHitRatio[dbName], err = models.GetIndexCacheHitRatio(db); err != nil {
		return errors.Wrapf(err, "cannot get Index Cache Hit Ratio for the %s database", dbName)
	}

	return nil
}

func collect(db models.XODB, data *templateData) []error {
	errs := make([]error, 0)
	var err error

	version10, _ := version.NewVersion("10.0")

	ch := make(chan interface{}, 2)
	getCounters(db, ch)
	c1, err := waitForCounters(ch)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "Cannot get counters (1st run)"))
	} else {
		for _, counters := range c1 {
			data.Counters[counters.Datname] = append(data.Counters[counters.Datname], counters)
		}
	}

	go func() {
		time.Sleep(time.Duration(data.Sleep) * time.Second)
		getCounters(db, ch)
	}()

	if data.ClusterInfo, err = models.GetClusterInfos(db); err != nil {
		errs = append(errs, errors.Wrap(err, "Cannot get cluster info"))
	}

	if data.ConnectedClients, err = models.GetConnectedClients(db); err != nil {
		errs = append(errs, errors.Wrap(err, "Cannot get the connected clients list"))
	}

	if data.DatabaseWaitEvents, err = models.GetDatabaseWaitEvents(db); err != nil {
		errs = append(errs, errors.Wrap(err, "Cannot get databases wait events"))
	}

	if data.GlobalWaitEvents, err = models.GetGlobalWaitEvents(db); err != nil {
		errs = append(errs, errors.Wrap(err, "Cannot get Global Wait Events"))
	}

	if data.PortAndDatadir, err = models.GetPortAndDatadir(db); err != nil {
		errs = append(errs, errors.Wrap(err, "Cannot get Port and Dir"))
	}

	if data.Tablespaces, err = models.GetTablespaces(db); err != nil {
		errs = append(errs, errors.Wrap(err, "Cannot get Tablespaces"))
	}

	if data.SlaveHosts96, err = models.GetSlaveHosts96s(db); err != nil {
		errs = append(errs, errors.Wrap(err, "Cannot get slave hosts on Postgre < 10"))
	}

	if !data.ServerVersion.LessThan(version10) {
		if data.SlaveHosts10, err = models.GetSlaveHosts10s(db); err != nil {
			errs = append(errs, errors.Wrap(err, "Cannot get slave hosts in Postgre 10+"))
		}
	}

	c2, err := waitForCounters(ch)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "Cannot read counters (2nd run)"))
	} else {
		for _, counters := range c2 {
			data.Counters[counters.Datname] = append(data.Counters[counters.Datname], counters)
		}
		calcCountersDiff(data.Counters)
	}

	return errs
}

func getCounters(db models.XODB, ch chan interface{}) {
	counters, err := models.GetCounters(db)
	if err != nil {
		ch <- err
	} else {
		ch <- counters
	}
}

func waitForCounters(ch chan interface{}) ([]*models.Counters, error) {
	resp := <-ch
	if err, ok := resp.(error); ok {
		return nil, err
	}

	return resp.([]*models.Counters), nil
}

func parseServerVersion(v string) (*version.Version, error) {
	re := regexp.MustCompile(`(\d?\d)(\d\d)(\d\d)`)
	m := re.FindStringSubmatch(v)
	if len(m) != 4 {
		return nil, fmt.Errorf("cannot parse version %s", v)
	}
	return version.NewVersion(fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3]))
}
func calcCountersDiff(counters map[models.Name][]*models.Counters) {
	for dbName, c := range counters {
		diff := &models.Counters{
			Datname:      dbName,
			Numbackends:  c[1].Numbackends - c[0].Numbackends,
			XactCommit:   c[1].XactCommit - c[0].XactCommit,
			XactRollback: c[1].XactRollback - c[0].XactRollback,
			BlksRead:     c[1].BlksRead - c[0].BlksRead,
			BlksHit:      c[1].BlksHit - c[0].BlksHit,
			TupReturned:  c[1].TupReturned - c[0].TupReturned,
			TupFetched:   c[1].TupFetched - c[0].TupFetched,
			TupInserted:  c[1].TupInserted - c[0].TupInserted,
			TupUpdated:   c[1].TupUpdated - c[0].TupUpdated,
			TupDeleted:   c[1].TupDeleted - c[0].TupDeleted,
			Conflicts:    c[1].Conflicts - c[0].Conflicts,
			TempFiles:    c[1].TempFiles - c[0].TempFiles,
			TempBytes:    c[1].TempBytes - c[0].TempBytes,
			Deadlocks:    c[1].Deadlocks - c[0].Deadlocks,
		}
		counters[dbName] = append(counters[dbName], diff)
	}
}

func parseCommandLineOpts(args []string) (cliOptions, error) {
	var version, commit string
	app := kingpin.New("pt-pg-summary", "Percona Toolkie - PostgreSQL Summary")
	app.Version(fmt.Sprintf("%s version %s, git commit %s", app.Name, version, commit))
	opts := cliOptions{}
	app.Flag("all-databases", "summarize all databases").BoolVar(&opts.AllDatabases)
	app.Flag("ask-pass", "Prompt for a password when connecting to PostgreSQL").BoolVar(&opts.AskPass)
	app.Flag("config", "Config file").StringVar(&opts.Config)
	app.Flag("databases", "Summarize this comma-separated list of databases").StringsVar(&opts.Databases)
	app.Flag("defaults-file", "Only read PostgreSQL options from the given file").StringVar(&opts.DefaultsFile)
	app.Flag("host", "Host to connect to").StringVar(&opts.Host)
	app.Flag("list-encrypted-tables", "Include a list of the encrypted tables in all databases").
		Hidden().BoolVar(&opts.ListEncryptedTables)
	app.Flag("password", "Password to use when connecting").StringVar(&opts.Password)
	app.Flag("port", "Port number to use for connection").IntVar(&opts.Port)
	app.Flag("read-samples", "Create a report from the files found in this directory").StringVar(&opts.ReadSamples)
	app.Flag("save-samples", "Save the data files used to generate the summary in this directory").
		StringVar(&opts.SaveSamples)
	app.Flag("sleep", "Seconds to sleep when gathering status counters").IntVar(&opts.Seconds)
	app.Flag("socket", "ocket file to use for connection").StringVar(&opts.Socket)
	app.Flag("username", "User for login if not current user").StringVar(&opts.User)
	app.Flag("disable-ssl", "Diable SSL for the connection").Default("true").BoolVar(&opts.DisableSSL)

	_, err := app.Parse(args)
	return opts, err
}
