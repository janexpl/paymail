package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/janexpl/paymail/internal"
)

func main() {
	configPath := flag.String("config", "", "Config path")
	employeePath := flag.String("employee", "", "Employee list path")
	workers := flag.Int("workers", 4, "Number of concurrent email senders")
	flag.Parse()

	if *configPath == "" || *employeePath == "" {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(*configPath, *employeePath, *workers); err != nil {
		log.Fatal(err)
	}
}

func run(configPath, employeePath string, workers int) error {
	if workers < 1 {
		return fmt.Errorf("workers must be greater than zero")
	}

	config, err := internal.NewConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	employees, err := internal.NewEmployeeDirectory(employeePath)
	if err != nil {
		return fmt.Errorf("load employees: %w", err)
	}

	conn, err := sql.Open("mssql", config.Database.ConnectionString())
	if err != nil {
		return fmt.Errorf("open database connection: %w", err)
	}
	defer conn.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelPing()
	if err := conn.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	controller := internal.NewPaymentsController(conn, employees, config)
	sendCtx, cancelSend := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelSend()

	if err := controller.SendOverduePaymentEmails(sendCtx, workers); err != nil {
		return fmt.Errorf("send overdue payment emails: %w", err)
	}

	return nil
}
