package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Backup menjalankan dump dan menulis hasilnya ke outDir.
// File ditulis ke *.tmp dulu, lalu di-rename setelah sukses,
// sehingga tidak ada file backup setengah jadi.
func Backup(cfg DBConfig, outDir string, compress bool) (string, int64, error) {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return "", 0, fmt.Errorf("buat folder output: %w", err)
	}

	name := fmt.Sprintf("%s_%s_%s.sql", cfg.Type, cfg.Name, time.Now().Format("20060102_150405"))
	if compress {
		name += ".gz"
	}
	final := filepath.Join(outDir, name)
	tmp := final + ".tmp"

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("buat file: %w", err)
	}
	ok := false
	defer func() {
		if !ok {
			f.Close()
			os.Remove(tmp)
		}
	}()

	var w io.Writer = f
	var gz *gzip.Writer
	if compress {
		gz = gzip.NewWriter(f)
		w = gz
	}

	cmd := cfg.DumpCmd()
	var stderr bytes.Buffer
	cmd.Stdout = w
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("%s gagal: %v: %s", cmd.Args[0], err, strings.TrimSpace(stderr.String()))
	}

	if gz != nil {
		if err := gz.Close(); err != nil {
			return "", 0, fmt.Errorf("tutup gzip: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		return "", 0, fmt.Errorf("tutup file: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		return "", 0, fmt.Errorf("rename file: %w", err)
	}
	ok = true

	info, err := os.Stat(final)
	if err != nil {
		return final, 0, nil
	}
	return final, info.Size(), nil
}

// Restore membaca file backup (.sql atau .sql.gz) dan mengirimnya ke psql/mysql.
func Restore(cfg DBConfig, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("buka file: %w", err)
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(file, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("baca gzip: %w", err)
		}
		defer gz.Close()
		r = gz
	}

	cmd := cfg.RestoreCmd()
	var stderr bytes.Buffer
	cmd.Stdin = r
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s gagal: %v: %s", cmd.Args[0], err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// TestConnection memastikan database bisa diakses.
func TestConnection(cfg DBConfig) error {
	cmd := cfg.PingCmd()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("koneksi gagal: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
