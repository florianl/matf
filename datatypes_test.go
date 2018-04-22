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
		name string
		data *[]byte
		order binary.ByteOrder
		dataType int
		numberOfBytes int
		ele interface{}
		err  string
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
