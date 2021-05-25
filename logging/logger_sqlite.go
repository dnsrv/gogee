package logging

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"sync"
	"time"
)

var (
	conn         *sql.DB
	stmt         *sql.Stmt
	mutex        = sync.Mutex{}
	buffer       []row
	flushTimeout time.Duration
	cancelChan   = make(chan int)
	l            = log.New(os.Stderr, "", log.LstdFlags)
)

type LoggerSqlite struct {
	prefix string
}

func NewLoggerSqlite(ctx context.Context, wg *sync.WaitGroup, dbFilename string, ft time.Duration) LoggerInterface {

	flushTimeout = ft

	conn, err := sql.Open("sqlite3", dbFilename)
	if err != nil {
		log.Fatalf("[logger] can't open file: %s", err)
	}

	s, err := prepareStatement(conn)
	if err != nil {
		log.Fatal(err)
	}
	stmt = s

	go ctxListener(ctx, wg)
	go flushChecker(ctx, wg)

	logger := &LoggerSqlite{}

	return logger
}

func (o *LoggerSqlite) Info(text string) {
	o.insert(LogLevelPrefixInfo, text)
	l.Println(LogLevelPrefixInfo + text)
}

func (o *LoggerSqlite) Error(text string) {
	o.insert(LogLevelPrefixWarn, text)
	l.Println(LogLevelPrefixWarn + text)
}

func (o *LoggerSqlite) Fatal(text string) {
	o.insert(LogLevelPrefixFatal, text)
	l.Println(LogLevelPrefixFatal + text)
	os.Exit(1)
}

func (o *LoggerSqlite) WithPrefix(p string) LoggerInterface {
	return &LoggerSqlite{
		prefix: p,
	}
}

func (o *LoggerSqlite) Close() {
	l.Println("[logger] closing logger")
	cancelChan <- 1
}

func (o *LoggerSqlite) insert(level, text string) {
	rowEntry := row{
		prefix: o.prefix,
		level:  level,
		text:   text,
		time:   time.Now(),
	}

	buffer = append(buffer, rowEntry)
}

// ----------------------------------------------------------------------------

func flush(c chan<- int, data []row) {

	_, _ = stmt.Exec("logger", LogLevelPrefixInfo,
		fmt.Sprintf("[logger] flushing logs: %d", len(data)), time.Now())

	for _, r := range data {
		_, err := stmt.Exec(r.prefix, r.level, r.text, r.time)
		if err != nil {
			l.Println("[logger][flush] log insert failed: ")
		}
	}
	l.Println("[logger][flush] flushing logs: finish")
	c <- 1
}

func flushChecker(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	ticker := time.NewTicker(flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-cancelChan:
			mutex.Lock()
			bufferCopy := buffer[0:]
			buffer = buffer[:0]
			mutex.Unlock()

			flushChan := make(chan int)
			go flush(flushChan, bufferCopy)
			<-flushChan
			_ = conn.Close()

			wg.Done()
			return
		case <-ticker.C:
			l.Println("[logger][flushChecker] checking new logs to flush")
			if len(buffer) > 0 {

				mutex.Lock()
				bufferCopy := buffer[0:]
				buffer = buffer[:0]
				mutex.Unlock()

				flushChan := make(chan int)
				go flush(flushChan, bufferCopy)
				<-flushChan
			}
		}
	}
}

func ctxListener(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	for {
		select {
		case <-ctx.Done():
			cancelChan <- 1
			wg.Done()
			return
		}
	}
}

// ----------------------------------------------------------------------------

func prepareStatement(conn *sql.DB) (*sql.Stmt, error) {

	if err := conn.Ping(); err != nil {
		return nil, errors.New("[logger] ping error: " + err.Error())
	}

	_, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			id 			INTEGER PRIMARY KEY AUTOINCREMENT, 
			prefix		VARCHAR(256) NULL,
			level       CHAR(256) NOT NULL,
			description VARCHAR(256) NOT NULL,
			is_exported INT NULL,
			created_at 	DATETIME     NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)

	if err != nil {
		return nil, errors.New("[logger] can not create logs table: " + err.Error())
	}

	stmt, err := conn.Prepare(
		"INSERT INTO logs (prefix, level, description, created_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return nil, errors.New("[logger] can not create prepared statement: " + err.Error())
	}

	return stmt, nil
}

type row struct {
	prefix string
	level  string
	text   string
	time   time.Time
}

type cache []row

func (o *cache) append(row row) {
	*o = append(*o, row)
}
