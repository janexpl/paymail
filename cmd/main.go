package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"runtime"
	"sync"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/janexpl/paymail/internal"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}
func main() {
	configPtr := flag.String("config", "", "Config path")
	employeePtr := flag.String("employee", "", "Employee list path")
	flag.Parse()
	config, _ := internal.NewConfig(*configPtr)
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d",
		config.Database.Host,
		config.Database.Username,
		config.Database.Password,
		config.Database.Port)

	conn, err := sql.Open("mssql", connString)
	if err != nil {
		log.Fatal("Open connection failed:", err.Error())
	}
	defer conn.Close()
	emp, _ := internal.NewEmployee(*employeePtr)
	emp.ReadAll()

	payments := internal.NewPayments(conn, emp, config)
	var wg sync.WaitGroup

	wg.Add(len(emp.Employees))
	for _, employee := range emp.Employees {
		go payments.SendEmail(employee.Username, &wg)
	}
	wg.Wait()
}
