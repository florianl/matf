package matf

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"testing"
)

func TestExtractDataElement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		data          *[]byte
		order         binary.ByteOrder
		dataType      int
		numberOfBytes int
		ele           interface{}
		err           string
	}{
		{name: "Unknown", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: 42, numberOfBytes: 42, err: "is not supported"},
		{name: "MiInt8", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiInt8, numberOfBytes: 1, ele: 17},
		{name: "MiUint8", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiUint8, numberOfBytes: 1, ele: 17},
		{name: "MiInt16", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiInt16, numberOfBytes: 2, ele: 8721},
		{name: "MiUint16", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiUint16, numberOfBytes: 2, ele: 8721},
		{name: "MiInt32", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiInt32, numberOfBytes: 4, ele: 1144201745},
		{name: "MiUint32", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiUint32, numberOfBytes: 4, ele: 1144201745},
		{name: "MiInt32", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.BigEndian, dataType: MiInt32, numberOfBytes: 4, ele: 287454020},
		{name: "MiUint32", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.BigEndian, dataType: MiUint32, numberOfBytes: 4, ele: 287454020},
		{name: "MiInt64", data: &([]byte{0x11, 0x22, 0x33, 0x44, 0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiInt64, numberOfBytes: 8, ele: 4914309075945333265},
		{name: "MiUint64", data: &([]byte{0x11, 0x22, 0x33, 0x44, 0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiUint64, numberOfBytes: 8, ele: 4914309075945333265},
		{name: "MiDouble", data: &([]byte{0x11, 0x22, 0x33, 0x44, 0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiDouble, numberOfBytes: 8, ele: 3.529429556587807e+20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ele, err := extractDataElement(tc.data, tc.order, tc.dataType, tc.numberOfBytes)
			if err != nil {
				if matched, _ := regexp.MatchString(tc.err, err.Error()); matched == false {
					t.Fatalf("Error matching regex: %v \t Got: %v", tc.err, err)
				}
			} else if len(tc.err) != 0 {
				t.Fatalf("Expected error, got none")
			}
			fmt.Println(ele, tc.ele)
		})
	}
}

func TestExtractNumeric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		data  *[]byte
		order binary.ByteOrder
		step  int
		ele   interface{}
		err   string
	}{
		{name: "[1,2]", data: &([]byte{0x09, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40}), order: binary.LittleEndian, step: 24, ele: []int{1, 2}},
		{name: "SmallData", data: &([]byte{0x06, 0x00, 0x04, 0x00, 0x01, 0x03, 0x03, 0x07}), order: binary.LittleEndian, step: 8, ele: []int{117637889}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ele, step, err := extractNumeric(tc.data, tc.order)
			if err != nil {
				if matched, _ := regexp.MatchString(tc.err, err.Error()); matched == false {
					t.Fatalf("Error matching regex: %v \t Got: %v", tc.err, err)
				}
			} else if len(tc.err) != 0 {
				t.Fatalf("Expected error, got none")
			}
			fmt.Println(ele, tc.ele, "\t", step, tc.step)
		})
	}
}

func TestExtractFieldNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		data            *[]byte
		fieldNameLength int
		numberOfFields  int
		fields          []string
		err             string
	}{
		{name: "['abc']", data: &([]byte{0x61, 0x62, 0x63}), fieldNameLength: 3, numberOfFields: 1, fields: []string{"abc"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fields, err := extractFieldNames(tc.data, tc.fieldNameLength, tc.numberOfFields)
			if err != nil {
				if matched, _ := regexp.MatchString(tc.err, err.Error()); matched == false {
					t.Fatalf("Error matching regex: %v \t Got: %v", tc.err, err)
				}
			} else if len(tc.err) != 0 {
				t.Fatalf("Expected error, got none")
			}
			fmt.Println(fields, tc.fields)
		})
	}
}

func TestExtractArrayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		data      *[]byte
		order     binary.ByteOrder
		arrayName string
		step      int
		err       string
	}{
		{name: "ThisIsALongerName", data: &([]byte{0x01, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x54, 0x68, 0x69, 0x73, 0x49, 0x73, 0x41, 0x4c, 0x6f, 0x6e, 0x67, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), order: binary.LittleEndian, step: 25, arrayName: "ThisIsALongerName"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, step, err := extractArrayName(tc.data, tc.order)
			if err != nil {
				if matched, _ := regexp.MatchString(tc.err, err.Error()); matched == false {
					t.Fatalf("Error matching regex: %v \t Got: %v", tc.err, err)
				}
			} else if len(tc.err) != 0 {
				t.Fatalf("Expected error, got none")
			}
			fmt.Println(name, tc.arrayName, "\t", step, tc.step)
		})
	}
}
