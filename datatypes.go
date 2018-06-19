package matf

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/pkg/errors"
)

// List of all MAT-File Data Types
const (
	MiInt8       int = 1
	MiUint8      int = 2
	MiInt16      int = 3
	MiUint16     int = 4
	MiInt32      int = 5
	MiUint32     int = 6
	MiSingle     int = 7
	MiDouble     int = 9
	MiInt64      int = 12
	MiUint64     int = 13
	MiMatrix     int = 14
	MiCompressed int = 15
	MiUtf8       int = 16
	MiUtf16      int = 17
	MiUtf32      int = 18
)

// List of all MAT-File Array Types
const (
	MxCellClass   int = 1
	MxStructClass int = 2
	MxObjectClass int = 3
	MxCharClass   int = 4
	MxSparseClass int = 5
	MxDoubleClass int = 6
	MxSingleClass int = 7
	MxInt8Class   int = 8
	MxUint8Class  int = 9
	MxInt16Class  int = 10
	MxUint16Class int = 11
	MxInt32Class  int = 12
	MxUint32Class int = 13
	MxInt64Class  int = 14
	MxUint64Class int = 15
)

func extractDataElement(r io.Reader, order binary.ByteOrder, dataType, numberOfBytes int) (interface{}, int, error) {
	var element interface{}
	var elements []interface{}
	var err error
	var data []byte
	var i int

	if dataType == MiMatrix {
		element, i, err = extractMatrix(r, order)
		if err != nil {
			return nil, 0, errors.Wrap(err, "\nextractMatrix() in extractDataElement() failed")
		}
		return element, i, nil
	}

	data, err = readMatfBytes(r, order, numberOfBytes)

	if err != nil {
		return nil, 0, errors.Wrap(err, "\nreadMatfBytes() in extractDataElement() failed")
	}
	for i < numberOfBytes {
		switch dataType {
		case MiInt8:
			element = int8(data[i])
			i++
		case MiUint8:
			element = uint8(data[i])
			i++
		case MiInt16:
			element = int16(order.Uint16(data[i : i+2]))
			i += 2
		case MiUint16:
			element = order.Uint16(data[i : i+2])
			i += 2
		case MiInt32:
			element = int32(order.Uint32(data[i : i+4]))
			i += 4
		case MiUint32:
			element = order.Uint32(data[i : i+4])
			i += 4
		case MiSingle:
			bits := order.Uint32(data[i : i+4])
			element = math.Float32frombits(bits)
			i += 4
		case MiInt64:
			element = int64(order.Uint64(data[i : i+8]))
			i += 8
		case MiUint64:
			element = order.Uint64(data[i : i+8])
			i += 8
		case MiDouble:
			bits := order.Uint64(data[i : i+8])
			element = math.Float64frombits(bits)
			i += 8
		default:
			return nil, 0, fmt.Errorf("Data Type %d is not supported", dataType)
		}
		elements = append(elements, element)
	}
	return elements, i, nil
}

func extractNumeric(r io.Reader, order binary.ByteOrder) (interface{}, int, error) {
	dataType, numberOfBytes, offset, err := extractTag(r, order)
	if err != nil {
		return nil, 0, errors.Wrap(err, "\nextractTag() in extractNumeric() failed")
	}

	re, _, err := extractDataElement(r, order, int(dataType), int(numberOfBytes))
	if err != nil {
		return nil, 0, errors.Wrap(err, "\nextractDataElement() in extractNumeric() failed")
	}
	return re, offset + int(numberOfBytes), err
}

func extractFieldNames(r io.Reader, order binary.ByteOrder, fieldNameLength, numberOfFields int) ([]string, error) {
	var index int
	var names []string
	if fieldNameLength*numberOfFields == 0 {
		return names, nil
	}
	data, err := readMatfBytes(r, order, fieldNameLength*numberOfFields)
	if err != nil {
		return []string{}, errors.Wrap(err, "\nreadMatfBytes() in extractFieldNames() failed")
	}
	for ; numberOfFields > 0; numberOfFields-- {
		str := string(data[index : index+fieldNameLength])
		names = append(names, str)
		index += fieldNameLength
	}
	return names, nil
}

func extractArrayName(r io.Reader, order binary.ByteOrder) (string, int, error) {
	_, numberOfBytes, offset, err := extractTag(r, order)
	if err != nil {
		return "", 0, errors.Wrap(err, "\nextractTag() in extractArrayName() failed")
	}
	if numberOfBytes == 0 {
		return "", offset, nil
	}

	arrayName, err := readMatfBytes(r, order, int(numberOfBytes))
	if err != nil {
		return "", offset, fmt.Errorf("Unable to read %d bytes: %v", numberOfBytes, err)
	}

	return string(arrayName), offset + int(numberOfBytes), nil
}

func extractTag(r io.Reader, order binary.ByteOrder) (uint32, uint32, int, error) {
	var dataType, numberOfBytes uint32
	var offset int

	data, err := readMatfBytes(r, order, 4)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Unable to read %d bytes: %v", 4, err)
	}
	// Small Data Element
	if (data[2] != data[3]) && order == binary.LittleEndian {
		dataType = uint32(order.Uint16(data[0:2]))
		numberOfBytes = uint32(order.Uint16(data[2:4]))
		offset = 4
	} else {
		extend, err := readMatfBytes(r, order, 4)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("Unable to read %d bytes: %v", 4, err)
		}
		dataType = order.Uint32(data)
		numberOfBytes = order.Uint32(extend)
		offset = 8
	}

	return dataType, numberOfBytes, offset, nil
}

func readMatfBytes(r io.Reader, order binary.ByteOrder, numberOfBytes int) ([]byte, error) {
	if numberOfBytes == 0 {
		return nil, fmt.Errorf("readMatfBytes(): will not read 0 bytes")
	}
	data := make([]byte, numberOfBytes)
	err := binary.Read(r, order, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
