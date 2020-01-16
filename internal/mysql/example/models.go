// Code generated by sqlc. DO NOT EDIT.

package teachersDB

import (
	"database/sql"
)

type DepartmentType string

const (
	English DepartmentType = "English"
	Math    DepartmentType = "Math"
)

func (e *DepartmentType) Scan(src interface{}) error {
	*e = DepartmentType(src.([]byte))
	return nil
}

type Teacher struct {
	ID         int             `json:"id"`
	FirstName  sql.NullString  `json:"first_name"`
	LastName   sql.NullString  `json:"last_name"`
	SchoolID   int             `json:"school_id"`
	ClassID    int             `json:"class_id"`
	SchoolLat  sql.NullFloat64 `json:"school_lat"`
	SchoolLng  sql.NullFloat64 `json:"school_lng"`
	Department DepartmentType  `json:"department"`
}

type Student struct {
	ID        int            `json:"id"`
	ClassID   int            `json:"class_id"`
	FirstName sql.NullString `json:"first_name"`
	LastName  sql.NullString `json:"last_name"`
}
