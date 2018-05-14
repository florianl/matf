package matf

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var (
	verySimpleMatrix = []byte{0x06, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x4d, 0x61, 0x54, 0x72, 0x49, 0x78, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x48, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x3f}
)

func TestExtractDataElement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		data          *[]byte
		order         binary.ByteOrder
		dataType      int
		numberOfBytes int
		step          int
		ele           interface{}
		err           string
	}{
		{name: "Unknown", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: 42, numberOfBytes: 42, step: 0, err: "is not supported"},
		{name: "MiInt8", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiInt8, numberOfBytes: 1, step: 1, ele: []interface{}{17}},
		{name: "MiUint8", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiUint8, numberOfBytes: 1, step: 1, ele: 17},
		{name: "MiInt16", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiInt16, numberOfBytes: 2, step: 2, ele: 8721},
		{name: "MiUint16", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiUint16, numberOfBytes: 2, step: 2, ele: 8721},
		{name: "MiInt32", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiInt32, numberOfBytes: 4, step: 4, ele: 1144201745},
		{name: "MiUint32", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiUint32, numberOfBytes: 4, step: 4, ele: 1144201745},
		{name: "MiInt32", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.BigEndian, dataType: MiInt32, numberOfBytes: 4, step: 4, ele: 287454020},
		{name: "MiUint32", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.BigEndian, dataType: MiUint32, numberOfBytes: 4, step: 4, ele: 287454020},
		{name: "MiSingle", data: &([]byte{0x11, 0x22, 0x33, 0x44}), order: binary.BigEndian, dataType: MiSingle, numberOfBytes: 4, step: 4, ele: 1.2795344e-28},
		{name: "MiInt64", data: &([]byte{0x11, 0x22, 0x33, 0x44, 0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiInt64, numberOfBytes: 8, step: 8, ele: 4914309075945333265},
		{name: "MiUint64", data: &([]byte{0x11, 0x22, 0x33, 0x44, 0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiUint64, numberOfBytes: 8, step: 8, ele: 4914309075945333265},
		{name: "MiDouble", data: &([]byte{0x11, 0x22, 0x33, 0x44, 0x11, 0x22, 0x33, 0x44}), order: binary.LittleEndian, dataType: MiDouble, numberOfBytes: 8, step: 8, ele: 3.529429556587807e+20},
		{name: "MiMatrix", data: &(verySimpleMatrix), order: binary.LittleEndian, dataType: MiMatrix, numberOfBytes: 1, step: 128, ele: []interface{}{MatMatrix{Name: "MaTrIx", Flags: 0x6, Class: 0x6, Dimensions: Dimensions{X: 3, Y: 3, Z: 0}, Content: NumPrt{RealPart: []interface{}{1, 0, 1, 0, 1, 0, 1, 0, 1}, ImaginaryPart: interface{}(nil)}}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ele, step, err := extractDataElement(tc.data, tc.order, tc.dataType, tc.numberOfBytes)
			if err != nil {
				if matched, _ := regexp.MatchString(tc.err, err.Error()); matched == false {
					t.Fatalf("Error matching regex: %v \t Got: %v", tc.err, err)
				}
				return
			} else if len(tc.err) != 0 {
				t.Fatalf("Expected error, got none")
			}
			if step != tc.step {
				t.Fatalf("Step\tExpected: %d \t Got: %d", tc.step, step)
			}
			fmt.Printf("Expected: %#v\tGot: %#v\n", tc.ele, ele)
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
		{name: "TooFewBytes", data: &([]byte{0x01, 0x10}), order: binary.LittleEndian, step: 1, err: "Could not read bytes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ele, step, err := extractNumeric(tc.data, tc.order)
			if err != nil {
				if matched, _ := regexp.MatchString(tc.err, err.Error()); matched == false {
					t.Fatalf("Error matching regex: %v \t Got: %v", tc.err, err)
				}
				return
			} else if len(tc.err) != 0 {
				t.Fatalf("Expected error, got none")
			}
			if step != tc.step {
				t.Fatalf("Step\tExpected: %d \t Got: %d", tc.step, step)
			}
			fmt.Printf("Expected: %#v\tGot: %#v\n", tc.ele, ele)
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
				return
			} else if len(tc.err) != 0 {
				t.Fatalf("Expected error, got none")
			}
			if !reflect.DeepEqual(tc.fields, fields) {
				t.Fatalf("Fields\tExpected: %#v\tGot: %#v\n", tc.fields, fields)
			}
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
		{name: "TooFewBytes", data: &([]byte{0x01, 0x10}), order: binary.LittleEndian, step: 1, err: "Could not read bytes"},
		{name: "ZeroLength", data: &([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}), order: binary.LittleEndian, step: 8, arrayName: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, step, err := extractArrayName(tc.data, tc.order)
			if err != nil {
				if matched, _ := regexp.MatchString(tc.err, err.Error()); matched == false {
					t.Fatalf("Error matching regex: %v \t Got: %v", tc.err, err)
				}
				return
			} else if len(tc.err) != 0 {
				t.Fatalf("Expected error, got none")
			}
			if step != tc.step {
				t.Fatalf("Step\tExpected: %d \t Got: %d", tc.step, step)
			}
			if strings.Compare(name, tc.arrayName) != 0 {
				t.Fatalf("Fields\tExpected: %#v\tGot: %#v\n", tc.arrayName, name)
			}
		})
	}
}
