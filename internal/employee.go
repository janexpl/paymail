package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

// Employee describes a report recipient.
type Employee struct {
	Username string `json:"name"`
	Email    string `json:"email"`
}

// EmployeeDirectory stores employees and lookup indexes.
type EmployeeDirectory struct {
	Employees []Employee `json:"employees"`
	byName    map[string]string
}

// NewEmployeeDirectory loads employees from disk and validates the payload.
func NewEmployeeDirectory(path string) (*EmployeeDirectory, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseEmployeeDirectory(data)
}

// ParseEmployeeDirectory unmarshals and validates the employee list.
func ParseEmployeeDirectory(data []byte) (*EmployeeDirectory, error) {
	var directory EmployeeDirectory
	if err := json.Unmarshal(data, &directory); err != nil {
		return nil, err
	}

	directory.byName = make(map[string]string, len(directory.Employees))
	for idx, employee := range directory.Employees {
		employee.Username = strings.TrimSpace(employee.Username)
		employee.Email = strings.TrimSpace(employee.Email)

		if employee.Username == "" {
			return nil, fmt.Errorf("employee at index %d is missing name", idx)
		}
		if employee.Email == "" {
			return nil, fmt.Errorf("employee %q is missing email", employee.Username)
		}
		if _, exists := directory.byName[employee.Username]; exists {
			return nil, fmt.Errorf("duplicate employee name %q", employee.Username)
		}

		directory.Employees[idx] = employee
		directory.byName[employee.Username] = employee.Email
	}

	return &directory, nil
}

// EmailByName returns employee email for a given username.
func (dir *EmployeeDirectory) EmailByName(name string) (string, bool) {
	if dir == nil {
		return "", false
	}

	email, ok := dir.byName[name]
	return email, ok
}
