package main

import (
	"bufio"
	"database/sql"
	"errors"
	"flag"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DorkBusterIP  = "23.227.170.221"
	Server        = flag.String("s", "", "The server we're working with. Required.")
	Database      = flag.String("d", "q2logs.sqlite", "The SQLite3 database file")
	Verbose       = flag.Bool("v", false, "Show more")
	Write         = flag.Bool("w", false, "Actually write data to the database")
	Logfile       = flag.String("f", "", "The log file to parse. Required")
	Chats         = []ChatLogEntry{}
	Privs         = []PrivmsgEntry{}
	Connects      = []ConnectEntry{}
	Renames       = []RenameEntry{}
	Rcons         = []RconEntry{}
	ConnectRegexp = &regexp.Regexp{}
	RenameRegexp  = &regexp.Regexp{}
	PrivmsgRegexp = &regexp.Regexp{}
	RconRegexp1   = &regexp.Regexp{}
	RconRegexp2   = &regexp.Regexp{}
	err           error
	DB            *sql.DB
	MidRcon       = false // still need to parse the rest of rcon command
)

type LogEntry struct {
	Timestamp int64 // a unix timestamp
	Context   string
	Entry     string
}

// these are broken up into 2 consecutive log lines
type RconEntry struct {
	Timestamp int64
	IP        string
	Limited   bool
	Invalid   bool
	Command   string
}

type ChatLogEntry struct {
	Timestamp int64
	Name      string
	Content   string
	Team      bool
}

type PrivmsgEntry struct {
	Timestamp int64
	Name1     string
	Name2     string
	Content   string
}

type ConnectEntry struct {
	Timestamp int64
	Name      string
	IP        string
	Client    string
}

type RenameEntry struct {
	Timestamp int64
	Name1     string
	Name2     string
}

func main() {
	fp, err := os.Open(*Logfile)
	if err != nil {
		log.Fatal(err)
	}

	defer fp.Close()

	if *Verbose {
		log.Println("Parsing logs")
	}

	scanner := bufio.NewScanner(fp)
	start := time.Now()
	parsed := 0
	for scanner.Scan() {
		entry, err := ScanLine(scanner.Text())
		if err != nil {
			if *Verbose {
				log.Println("scan error", err)
			}
			continue
		}

		switch entry.Context {
		case "T":
			ParseChat(entry)
		case "A":
			ParseEntry(entry)
		}
		parsed++
	}
	duration := time.Since(start)
	if *Verbose {
		log.Println(parsed, "log lines parsed in", duration)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	WriteToDatabase()
}

func ScanLine(line string) (LogEntry, error) {
	if line == "" {
		return LogEntry{}, errors.New("blank line")
	}
	tokens := strings.SplitN(line, " ", 3)
	entry := LogEntry{
		Timestamp: LogDateToTimestamp(tokens[0]),
		Context:   tokens[1],
		Entry:     tokens[2],
	}

	return entry, nil
}

func ParseChat(e LogEntry) {
	// priv msg
	if PrivmsgRegexp.Match([]byte(e.Entry)) {
		result := PrivmsgRegexp.FindAllStringSubmatch(e.Entry, -1)
		pm := PrivmsgEntry{
			Timestamp: e.Timestamp,
			Name1:     result[0][1],
			Name2:     result[0][2],
			Content:   result[0][3],
		}
		Privs = append(Privs, pm)
		return
	}

	// regular chat
	tokens := strings.SplitN(e.Entry, ": ", 2)
	teamsay := false
	name := tokens[0]
	if (tokens[0])[0] == '(' {
		teamsay = true
		name = (tokens[0])[1 : len(tokens[0])-1]
	}
	chat := ChatLogEntry{
		Timestamp: e.Timestamp,
		Name:      name,
		Content:   tokens[1],
		Team:      teamsay,
	}

	Chats = append(Chats, chat)
}

// this is gross but really the only way to do it
func LogDateToTimestamp(ts string) int64 {
	if len(ts) != 15 {
		if *Verbose {
			log.Println("invalid time string:", ts)
		}
		return time.Now().Unix()
	}
	yr, _ := strconv.Atoi(ts[0:4])
	mt, _ := strconv.Atoi(ts[4:6])
	dy, _ := strconv.Atoi(ts[6:8])
	hr, _ := strconv.Atoi(ts[9:11])
	mn, _ := strconv.Atoi(ts[11:13])
	sc, _ := strconv.Atoi(ts[13:15])

	timestamp := time.Date(yr, time.Month(mt), dy, hr, mn, sc, 0, time.UTC)

	return timestamp.Unix()
}

// this shoud only be for generic messages, type "A"
func ParseEntry(e LogEntry) {
	// continuation of rcon, it's a 2 liner
	if MidRcon {
		MidRcon = false
		// remove TS rcon spam
		if e.Entry == "status" {
			if Rcons[len(Rcons)-1].IP == DorkBusterIP {
				Rcons = Rcons[:len(Rcons)-1] // remove last element
				return
			}
		}
		Rcons[len(Rcons)-1].Command = e.Entry
		return
	}

	if ConnectRegexp.Match([]byte(e.Entry)) {
		result := ConnectRegexp.FindAllStringSubmatch(e.Entry, -1)
		conn := ConnectEntry{
			Timestamp: e.Timestamp,
			Name:      result[0][1],
			IP:        result[0][2],
			Client:    result[0][3],
		}
		Connects = append(Connects, conn)
		return
	}

	if RenameRegexp.Match([]byte(e.Entry)) {
		result := RenameRegexp.FindAllStringSubmatch(e.Entry, -1)
		conn := RenameEntry{
			Timestamp: e.Timestamp,
			Name1:     result[0][1],
			Name2:     result[0][2],
		}
		Renames = append(Renames, conn)
		return
	}

	if RconRegexp1.Match([]byte(e.Entry)) {
		result := RconRegexp1.FindAllStringSubmatch(e.Entry, -1)
		rcon := RconEntry{
			Timestamp: e.Timestamp,
			Invalid:   result[0][1] != "",
			Limited:   result[0][2] != "",
			IP:        result[0][3],
		}
		Rcons = append(Rcons, rcon)
		MidRcon = true
		return
	}
}

func OpenDatabase() {
	db, err := sql.Open("sqlite3", *Database)
	if err != nil {
		log.Println(err)
		os.Exit(0)
	}
	DB = db
}

func CloseDatabase() {
	if err := DB.Close(); err != nil {
		log.Println(err)
	}
}

// Get the database ID for the given server string.
// If it doesn't exist, insert it.
func GetServerID(srv string) int {
	id := 0
	sqli := "SELECT id FROM server WHERE servername = ? LIMIT 1"
	if err := DB.QueryRow(sqli, srv).Scan(&id); err == nil {
		return id
	} else if err == sql.ErrNoRows {
		sqli = "INSERT INTO server (servername) VALUES (?)"
		res, errr := DB.Exec(sqli, srv)
		if errr != nil {
			log.Println(err)
			os.Exit(0)
		}
		if id, err := res.LastInsertId(); err != nil {
			log.Println(err)
			os.Exit(0)
		} else {
			return int(id)
		}
	} else {
		log.Println(err)
		os.Exit(0)
	}

	return 0
}

func WriteToDatabase() {
	// actually insert into the database
	if !*Write {
		return
	}

	total := 0

	if *Verbose {
		log.Println("Writing data")
	}

	OpenDatabase()
	sid := GetServerID(*Server)

	start := time.Now()
	sql := "INSERT INTO connect (timestamp, server, name, ip, client) VALUES (?,?,?,?,?)"
	for _, c := range Connects {
		_, err := DB.Exec(sql, c.Timestamp, sid, c.Name, c.IP, c.Client)
		if err != nil {
			log.Println(err)
		}
		total++
	}

	team := 0
	sql = "INSERT INTO chat (timestamp, server, name, team, msg) VALUES (?,?,?,?,?)"
	for _, c := range Chats {
		if c.Team {
			team = 1
		} else {
			team = 0
		}
		_, err := DB.Exec(sql, c.Timestamp, sid, c.Name, team, c.Content)
		if err != nil {
			log.Println(err)
		}
		total++
	}

	sql = "INSERT INTO chat_private (timestamp, server, name1, name2, msg) VALUES (?,?,?,?,?)"
	for _, pm := range Privs {
		_, err := DB.Exec(sql, pm.Timestamp, sid, pm.Name1, pm.Name2, pm.Content)
		if err != nil {
			log.Println(err)
		}
		total++
	}

	sql = "INSERT INTO rename (timestamp, server, name1, name2) VALUES (?,?,?,?)"
	for _, c := range Renames {
		_, err := DB.Exec(sql, c.Timestamp, sid, c.Name1, c.Name2)
		if err != nil {
			log.Println(err)
		}
		total++
	}

	sql = "INSERT INTO rcon (timestamp, server, ip, limited, invalid, command) VALUES (?,?,?,?,?,?)"
	for _, r := range Rcons {
		_, err := DB.Exec(sql, r.Timestamp, sid, r.IP, r.Limited, r.Invalid, r.Command)
		if err != nil {
			log.Println(err)
		}
		total++
	}
	duration := time.Since(start)
	CloseDatabase()

	if *Verbose {
		log.Println(total, "records written in", duration)
	}
}

func init() {
	flag.Parse()
	if *Logfile == "" {
		flag.Usage()
		os.Exit(0)
	}
	if *Server == "" {
		flag.Usage()
		os.Exit(0)
	}

	// name[1.1.1.1:22222]: q2pro version whatever
	ConnectRegexp, err = regexp.Compile(`^(?P<name>.+)\[(?P<ip>\d+\.\d+\.\d+\.\d+):\d+\]: (?P<client>.+)$`)
	if err != nil {
		log.Fatal(err)
	}

	// name1[1.1.1.1:22222] changed name to name2
	RenameRegexp, err = regexp.Compile(`^(?P<name1>.+)\[.+:\d+\] changed name to (?P<name2>.+)$`)
	if err != nil {
		log.Fatal(err)
	}

	// (name1)(private message to: name2) msg
	PrivmsgRegexp, err = regexp.Compile(`^\((?P<name1>.+)\)\(private message to: (?P<name2>.+)\) (?P<msg>.+)$`)
	if err != nil {
		log.Fatal(err)
	}

	RconRegexp1, err = regexp.Compile(`(?P<invalid>Invalid)?\s?(?P<limited>[Ll]imited)?\s?rcon from (?P<ip>\d+\.\d+\.\d+\.\d+):\d+:$`)
	if err != nil {
		log.Fatal(err)
	}
}
