package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// Employee structure
type Employee struct {
	Username string `json:"name"`
	Email    string `json:"email"`
}

// Employess structure with Employee array, and values of json query
type Employess struct {
	Employees []Employee `json:"employees"`
	values    []byte
}

// NewEmployee constructor - return pointer to Employees structure
func NewEmployee() (*Employess, error) {

	jsonFile, err := os.Open("../configs/employees.json")
	if err != nil {
		fmt.Println(err)
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	defer jsonFile.Close()

	return &Employess{
		values: byteValue,
	}, nil
}

// ReadAll - reading all records from json file
func (emp *Employess) ReadAll() (*Employess, error) {
	var employees Employess
	json.Unmarshal(emp.values, &employees)
	emp.Employees = employees.Employees
	return &employees, nil
}

// GetEmail - search email in structure at given name
func (emp *Employess) GetEmail(name string) string {
	for _, employee := range emp.Employees {
		if employee.Username == name {
			return employee.Email
		}
	}
	return ""
}
