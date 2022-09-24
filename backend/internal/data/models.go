package data

import (
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Spreadsheet SpreadsheetModel
}

func NewModels() Models {
	return Models{
		Spreadsheet: SpreadsheetModel{},
	}
}
