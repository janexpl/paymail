package main

import (
	"database/sql"
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

	config, _ := internal.NewConfig("../configs/config.yml")
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
	emp, _ := internal.NewEmployee()
	emp.ReadAll()

	payments := internal.NewPayments(conn, emp, config)
	var wg sync.WaitGroup

	wg.Add(len(emp.Employees))
	for _, employee := range emp.Employees {
		go payments.SendEmail(employee.Username, &wg)
	}
	wg.Wait()
}
