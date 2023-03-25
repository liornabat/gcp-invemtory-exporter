package xls

import (
	"fmt"
	excelize2 "github.com/360EntSecGroup-Skylar/excelize"
	"github.com/xuri/excelize/v2"
)

type Xls struct {
	file *excelize.File
}

func NewXls() *Xls {
	return &Xls{
		file: excelize.NewFile(),
	}
}

func (x *Xls) Close() error {
	return x.Close()
}

func (x *Xls) NewSheet(name string) error {
	_, err := x.file.NewSheet(name)
	if err != nil {
		return err
	}
	return nil
}

func (x *Xls) SetDataToSheet(sheet string, data [][]string) error {
	if err := x.NewSheet(sheet); err != nil {
		return err
	}
	for i, row := range data {
		for j, cell := range row {
			cellAxis := fmt.Sprintf("%s%d", excelize2.ToAlphaString(j), i+1)
			if err := x.file.SetCellValue(sheet, cellAxis, cell); err != nil {
				return err
			}
		}
	}
	return nil
}

func (x *Xls) DeleteSheet(sheet string) error {
	return x.file.DeleteSheet(sheet)
}
func (x *Xls) GetBytes() ([]byte, error) {
	buffer, err := x.file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
