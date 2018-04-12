package matf

import (
	"encoding/binary"
	"fmt"
	"math"
)

// MAT-File Data Types
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

// MAT-File Array Types
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

func extractDataElement(data *[]byte, order binary.ByteOrder, dataType, numberOfBytes int) (interface{}, error) {

	var element interface{}
	var elements []interface{}
	var err error

	for i := 0; i < numberOfBytes; {
		switch dataType {
		case MiMatrix:
			element, err = extractMatrix(*data, order)
			if err != nil {
				return nil, err
			}
		case MiInt8:
			element = int8((*data)[i])
			i++
		case MiUint8:
			element = uint8((*data)[i])
			i++
		case MiInt16:
			element = int16(order.Uint16((*data)[i:]))
			i += 2
		case MiUint16:
			element = order.Uint16((*data)[i:])
			i += 2
		case MiInt32:
			element = int32(order.Uint32((*data)[i:]))
			i += 4
		case MiUint32:
			element = order.Uint32((*data)[i:])
			i += 4
		case MiInt64:
			element = int64(order.Uint64((*data)[i:]))
			i += 8
		case MiUint64:
			element = order.Uint64((*data)[i:])
			i += 8
		case MiDouble:
			bits := order.Uint64((*data)[i:])
			element = math.Float64frombits(bits)
			i += 8
		default:
			return nil, fmt.Errorf("Data Type %d is not supported", dataType)
		}
		elements = append(elements, element)
	}
	return elements, nil
}
