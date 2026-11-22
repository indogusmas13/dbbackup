package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

const usage = `dbbackup - backup & restore MySQL / PostgreSQL

Pemakaian:
  dbbackup backup  [flags]   buat backup
  dbbackup restore [flags]   restore dari file backup
  dbbackup test    [flags]   tes koneksi database

Jalankan "dbbackup <perintah> -h" untuk daftar flag.
Config dibaca dari file .env (folder kerja atau folder executable),
lihat .env.example. Flag di command line menimpa nilai dari .env.
`

type common struct {
	cfg     DBConfig
	logFile string
	tgToken string
	tgChat  string
}

func registerCommon(fs *flag.FlagSet, c *common) {
	fs.StringVar(&c.cfg.Type, "type", os.Getenv("DB_TYPE"), "jenis database: postgres | mysql (env DB_TYPE)")
	fs.StringVar(&c.cfg.Host, "host", envOr("DB_HOST", "localhost"), "host database (env DB_HOST)")
	fs.IntVar(&c.cfg.Port, "port", envInt("DB_PORT"), "port, default 5432 postgres / 3306 mysql (env DB_PORT)")
	fs.StringVar(&c.cfg.User, "user", os.Getenv("DB_USER"), "user database (env DB_USER)")
	fs.StringVar(&c.cfg.Password, "password", "", "password (env DB_PASSWORD)")
	fs.StringVar(&c.cfg.Name, "db", os.Getenv("DB_NAME"), "nama database (env DB_NAME)")
	fs.StringVar(&c.logFile, "log", os.Getenv("LOG_FILE"), "path file log, opsional (env LOG_FILE)")
	fs.StringVar(&c.tgToken, "tg-token", "", "token bot Telegram (env TELEGRAM_BOT_TOKEN)")
	fs.StringVar(&c.tgChat, "tg-chat", os.Getenv("TELEGRAM_CHAT_ID"), "chat ID Telegram (env TELEGRAM_CHAT_ID)")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]

	if err := LoadDotEnv(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	var c common
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	registerCommon(fs, &c)

	var outDir, file string
	var compress bool
	switch cmd {
	case "backup":
		fs.StringVar(&outDir, "out", envOr("BACKUP_DIR", "./backups"), "folder tujuan backup (env BACKUP_DIR)")
		fs.BoolVar(&compress, "compress", true, "kompres dengan gzip")
	case "restore":
		fs.StringVar(&file, "file", "", "file backup .sql / .sql.gz (wajib)")
	case "test":
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "perintah tidak dikenal: %s\n\n%s", cmd, usage)
		os.Exit(2)
	}
	fs.Parse(args)

	// Nilai rahasia diisi setelah parse, supaya tidak tercetak di output -h.
	if c.cfg.Password == "" {
		c.cfg.Password = os.Getenv("DB_PASSWORD")
	}
	if c.tgToken == "" {
		c.tgToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	if c.cfg.Port == 0 {
		c.cfg.Port = defaultPort(c.cfg.Type)
	}
	if err := c.cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	logger, closeLog, err := NewLogger(c.logFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer closeLog()

	if err := run(cmd, c, logger, outDir, file, compress); err != nil {
		closeLog()
		os.Exit(1)
	}
}

func run(cmd string, c common, l *log.Logger, outDir, file string, compress bool) error {
	target := fmt.Sprintf("%s/%s@%s:%d", c.cfg.Type, c.cfg.Name, c.cfg.Host, c.cfg.Port)
	start := time.Now()
	var err error
	var msg string

	switch cmd {
	case "test":
		l.Printf("[TEST] %s", target)
		if err = TestConnection(c.cfg); err == nil {
			msg = fmt.Sprintf("koneksi OK: %s", target)
		}
	case "backup":
		l.Printf("[BACKUP] mulai %s", target)
		var path string
		var size int64
		if path, size, err = Backup(c.cfg, outDir, compress); err == nil {
			msg = fmt.Sprintf("backup sukses: %s -> %s (%d bytes, %s)", target, path, size, time.Since(start).Round(time.Millisecond))
		}
	case "restore":
		if file == "" {
			err = fmt.Errorf("-file wajib diisi")
			break
		}
		l.Printf("[RESTORE] mulai %s <- %s", target, file)
		if err = Restore(c.cfg, file); err == nil {
			msg = fmt.Sprintf("restore sukses: %s <- %s (%s)", target, file, time.Since(start).Round(time.Millisecond))
		}
	}

	if err != nil {
		msg = fmt.Sprintf("%s GAGAL (%s): %v", cmd, target, err)
		l.Printf("[ERROR] %s", msg)
	} else {
		l.Printf("[OK] %s", msg)
	}

	// Notifikasi Telegram hanya untuk backup & restore.
	if cmd != "test" {
		if nerr := NotifyTelegram(c.tgToken, c.tgChat, msg); nerr != nil {
			l.Printf("[WARN] notifikasi telegram gagal: %v", nerr)
		}
	}
	return err
}
