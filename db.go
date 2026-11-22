package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// DBConfig berisi parameter koneksi database.
type DBConfig struct {
	Type     string // "postgres" atau "mysql"
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

func (c DBConfig) Validate() error {
	if c.Type != "postgres" && c.Type != "mysql" {
		return fmt.Errorf("type harus 'postgres' atau 'mysql', didapat %q", c.Type)
	}
	if c.Name == "" {
		return fmt.Errorf("nama database (-db) wajib diisi")
	}
	if c.User == "" {
		return fmt.Errorf("user (-user) wajib diisi")
	}
	return nil
}

// env menambahkan password lewat environment variable,
// supaya password tidak terlihat di daftar proses (ps).
func (c DBConfig) env() []string {
	e := os.Environ()
	if c.Password == "" {
		return e
	}
	switch c.Type {
	case "postgres":
		e = append(e, "PGPASSWORD="+c.Password)
	case "mysql":
		e = append(e, "MYSQL_PWD="+c.Password)
	}
	return e
}

func (c DBConfig) connArgs() []string {
	port := strconv.Itoa(c.Port)
	switch c.Type {
	case "postgres":
		return []string{"-h", c.Host, "-p", port, "-U", c.User}
	default: // mysql
		return []string{"-h", c.Host, "-P", port, "-u", c.User}
	}
}

// DumpCmd: perintah untuk membuat backup (output ke stdout).
func (c DBConfig) DumpCmd() *exec.Cmd {
	var cmd *exec.Cmd
	switch c.Type {
	case "postgres":
		args := append(c.connArgs(), "--no-password", "--clean", "--if-exists", c.Name)
		cmd = exec.Command("pg_dump", args...)
	default:
		args := append(c.connArgs(), "--single-transaction", "--routines", "--triggers", c.Name)
		cmd = exec.Command("mysqldump", args...)
	}
	cmd.Env = c.env()
	return cmd
}

// RestoreCmd: perintah untuk restore (input SQL dari stdin).
func (c DBConfig) RestoreCmd() *exec.Cmd {
	var cmd *exec.Cmd
	switch c.Type {
	case "postgres":
		args := append(c.connArgs(), "--no-password", "-v", "ON_ERROR_STOP=1", "-q", "-d", c.Name)
		cmd = exec.Command("psql", args...)
	default:
		args := append(c.connArgs(), c.Name)
		cmd = exec.Command("mysql", args...)
	}
	cmd.Env = c.env()
	return cmd
}

// PingCmd: perintah untuk tes koneksi.
func (c DBConfig) PingCmd() *exec.Cmd {
	var cmd *exec.Cmd
	switch c.Type {
	case "postgres":
		args := append(c.connArgs(), "--no-password", "-d", c.Name, "-tAc", "SELECT 1")
		cmd = exec.Command("psql", args...)
	default:
		args := append(c.connArgs(), "-e", "SELECT 1", c.Name)
		cmd = exec.Command("mysql", args...)
	}
	cmd.Env = c.env()
	return cmd
}

func defaultPort(dbType string) int {
	if dbType == "mysql" {
		return 3306
	}
	return 5432
}
