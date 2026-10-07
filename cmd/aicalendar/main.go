package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Crashmere/AICalendar/internal/calendar"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"
)

func env(k, v string) string {
	if x := os.Getenv(k); x != "" {
		return x
	}
	return v
}
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: aicalendar init|serve|check|backup|restore|token")
	}
	cmd := os.Args[1]
	flags := flag.NewFlagSet(cmd, flag.ContinueOnError)
	db := flags.String("db", env("AICALENDAR_DB", "var/aicalendar.sqlite"), "SQLite path")
	out := flags.String("out", "", "new output file")
	from := flags.String("from", "", "backup source")
	listen := flags.String("listen", env("AICALENDAR_LISTEN", "127.0.0.1:18086"), "loopback listen address")
	base := flags.String("base-path", env("AICALENDAR_BASE_PATH", "/aicalendar/"), "public path")
	origin := flags.String("origin", os.Getenv("AICALENDAR_ORIGIN"), "public origin")
	tokenFile := flags.String("token-hash-file", os.Getenv("AICALENDAR_TOKEN_HASH_FILE"), "private SHA256 hash file")
	demo := flags.Bool("demo", false, "local development without configured public origin")
	if e := flags.Parse(os.Args[2:]); e != nil {
		return e
	}
	ctx := context.Background()
	switch cmd {
	case "token":
		if *out == "" {
			return fmt.Errorf("--out required; token will not be printed")
		}
		b := make([]byte, 32)
		if _, e := rand.Read(b); e != nil {
			return e
		}
		token := hex.EncodeToString(b)
		hash := sha256.Sum256([]byte(token))
		if e := writeNew(*out, []byte(token+"\n")); e != nil {
			return e
		}
		if e := writeNew(*out+".sha256", []byte(hex.EncodeToString(hash[:])+"\n")); e != nil {
			return e
		}
		fmt.Println("Token and SHA256 hash written to private files.")
		return nil
	case "check":
		return calendar.CheckFile(ctx, *db)
	case "restore":
		if *from == "" {
			return fmt.Errorf("--from required")
		}
		return calendar.Restore(ctx, *from, *db)
	case "init", "backup", "serve":
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
	store, e := calendar.Open(*db, cmd == "init")
	if e != nil {
		return e
	}
	defer store.Close()
	if cmd == "init" {
		fmt.Println("Initialized empty AICalendar database.")
		return nil
	}
	if cmd == "backup" {
		if *out == "" {
			return fmt.Errorf("--out required")
		}
		return store.Backup(ctx, *out)
	}
	host, _, e := net.SplitHostPort(*listen)
	if e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("serve must listen on an explicit loopback IP")
	}
	if !*demo && !calendar.ValidOrigin(*origin) {
		return fmt.Errorf("AICALENDAR_ORIGIN must be a valid public origin")
	}
	if !strings.HasPrefix(*base, "/") || !strings.HasSuffix(*base, "/") {
		return fmt.Errorf("base path must start and end with /")
	}
	hash := ""
	if *tokenFile != "" {
		b, e := os.ReadFile(*tokenFile)
		if e != nil {
			return e
		}
		hash = strings.TrimSpace(string(b))
		decoded, e := hex.DecodeString(hash)
		if e != nil || len(decoded) != 32 {
			return fmt.Errorf("invalid token hash")
		}
	}
	handler := (&calendar.Server{Store: store, Origin: *origin, BasePath: *base, TokenHash: hash, Demo: *demo}).Handler()
	server := &http.Server{Addr: *listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32768}
	stop, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-stop.Done()
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(c)
	}()
	msg, _ := json.Marshal(map[string]string{"listen": *listen, "base_path": *base})
	log.Printf("AICalendar started %s", msg)
	e = server.ListenAndServe()
	if e == http.ErrServerClosed {
		return nil
	}
	return e
}
func writeNew(p string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = f.Write(b)
	return e
}
func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
