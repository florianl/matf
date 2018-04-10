package matf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
)

// Flags
const (
	FlagComplex = 1 << 11
	FlagGlobal  = 1 << 10
	FlagLogical = 1 << 9
)

// Matf represents the MAT-file
type Matf struct {
	Header
	file         *os.File
	byteSwapping bool
}

// Dimensions contains the sizes of a MatMatrix
type Dimensions struct {
	x, y, z int
}

// MatMatrix represents a matrix
type MatMatrix struct {
	Name  string
	Flags uint32
	Dimensions
	RealPart      interface{}
	ImaginaryPart interface{}
}

// Header contains informations about the MAT-file
type Header struct {
	Text                string // 0 - 116
	SubsystemDataOffset []byte // 117 - 124
	Version             uint16 // 125 - 126
	EndianIndicator     uint16 // 127 - 128
}

func readHeader(mat *Matf, file *os.File) error {
	data := make([]byte, 128)
	count, err := file.Read(data)
	if err != nil {
		return err
	}

	if count != 128 {
		return fmt.Errorf("Could not read enough bytes")
	}

	mat.Header.Text = string(data[:116])
	mat.Header.SubsystemDataOffset = data[116:124]
	mat.Header.Version = binary.BigEndian.Uint16(data[124:126])
	mat.Header.EndianIndicator = binary.BigEndian.Uint16(data[126:128])

	if mat.Header.EndianIndicator == binary.BigEndian.Uint16([]byte{0x49, 0x4d}) {
		// EndianIndicator is IM rather than MI
		mat.byteSwapping = true
	}

	return nil
}

func readDimensions(data interface{}) (Dimensions, error) {
	var dim Dimensions
	t := reflect.ValueOf(data)

	for i := 0; i < t.Len(); i++ {
		value := reflect.ValueOf(t.Index(i).Interface()).Int()
		switch i {
		case 0:
			dim.x = int(value)
		case 1:
			dim.y = int(value)
		case 2:
			dim.z = int(value)
		default:
			return Dimensions{}, fmt.Errorf("More dimensions than exptected")
		}
	}
	return dim, nil
}

func isSmallDataElementFormat(data []byte, order binary.ByteOrder) (bool, error) {
	var offset int

	if order == binary.LittleEndian {
		offset = 2
	}

	small := make([]byte, 2)
	buf := bytes.NewReader(data[offset:])
	if err := binary.Read(buf, order, &small); err != nil {
		return false, fmt.Errorf("Could not read bytes:", err)
	}
	if small[0] != small[1] {
		// Small Data Element Format
		return true, nil
	}
	return false, nil
}

func checkIndex(index int) int {
	for {
		if index%8 == 0 {
			return index
		}
		index++
	}
}

func extractMatrix(data []byte, order binary.ByteOrder) (MatMatrix, error) {
	var matrix MatMatrix
	var index int
	var offset int
	var dataType uint32
	var numberOfBytes uint32
	var small bool
	var complexNumber bool
	var err error
	var buf *bytes.Reader

	// Array Flags
	small, err = isSmallDataElementFormat(data[index:], order)
	if err != nil {
		return MatMatrix{}, err
	}
	if small {
		//dataType = uint32(order.Uint16(data[index+0 : index+2]))
		numberOfBytes = uint32(order.Uint16(data[index+2 : index+4]))
		offset = 4
	} else {
		//dataType = order.Uint32(data[index+0 : index+4])
		numberOfBytes = order.Uint32(data[index+4 : index+8])
		offset = 8
	}
	arrayFlags := make([]byte, int(numberOfBytes))
	buf = bytes.NewReader(data[index+offset:])
	if err := binary.Read(buf, order, &arrayFlags); err != nil {
		return MatMatrix{}, err
	}
	matrix.Flags = binary.LittleEndian.Uint32(arrayFlags)
	if FlagComplex&matrix.Flags == FlagComplex {
		complexNumber = true
	}
	index += (offset + int(numberOfBytes))
	index = checkIndex(index)

	// Dimensions Array
	small, err = isSmallDataElementFormat(data[index:], order)
	if err != nil {
		return MatMatrix{}, err
	}
	if small {
		dataType = uint32(order.Uint16(data[index+0 : index+2]))
		numberOfBytes = uint32(order.Uint16(data[index+2 : index+4]))
		offset = 4
	} else {
		dataType = order.Uint32(data[index+0 : index+4])
		numberOfBytes = order.Uint32(data[index+4 : index+8])
		offset = 8
	}
	dimArray := make([]byte, int(numberOfBytes))
	buf = bytes.NewReader(data[index+offset:])
	if err := binary.Read(buf, order, &dimArray); err != nil {
		return MatMatrix{}, err
	}
	dims, err := extractDataElement(&dimArray, order, int(dataType), int(numberOfBytes))
	if err != nil {
		return MatMatrix{}, err
	}
	matrix.Dimensions, _ = readDimensions(dims)
	index += (offset + int(numberOfBytes))
	index = checkIndex(index)

	// Array Name
	small, err = isSmallDataElementFormat(data[index:], order)
	if err != nil {
		return MatMatrix{}, err
	}
	if small {
		//dataType = uint32(order.Uint16(data[index+0 : index+2]))
		numberOfBytes = uint32(order.Uint16(data[index+2 : index+4]))
		offset = 4
	} else {
		//dataType = order.Uint32(data[index+0 : index+4])
		numberOfBytes = order.Uint32(data[index+4 : index+8])
		offset = 8
	}
	arrayName := make([]byte, int(numberOfBytes))
	buf = bytes.NewReader(data[index+offset:])
	if err := binary.Read(buf, order, &arrayName); err != nil {
		return MatMatrix{}, err
	}
	matrix.Name = string(arrayName)
	index += (offset + int(numberOfBytes))
	index = checkIndex(index)

	// Real part
	small, err = isSmallDataElementFormat(data[index:], order)
	if err != nil {
		return MatMatrix{}, err
	}
	if small {
		dataType = uint32(order.Uint16(data[index+0 : index+2]))
		numberOfBytes = uint32(order.Uint16(data[index+2 : index+4]))
		offset = 4
	} else {
		dataType = order.Uint32(data[index+0 : index+4])
		numberOfBytes = order.Uint32(data[index+4 : index+8])
		offset = 8
	}
	tmp := data[index+offset:]
	re, err := extractDataElement(&tmp, order, int(dataType), int(numberOfBytes))
	if err != nil {
		return MatMatrix{}, err
	}
	matrix.RealPart = re
	index += (offset + int(numberOfBytes))
	index = checkIndex(index)

	// Imaginary part (optional)
	if complexNumber {
		small, err = isSmallDataElementFormat(data[index:], order)
		if err != nil {
			return MatMatrix{}, err
		}
		if small {
			dataType = uint32(order.Uint16(data[index+0 : index+2]))
			numberOfBytes = uint32(order.Uint16(data[index+2 : index+4]))
			offset = 4
		} else {
			dataType = order.Uint32(data[index+0 : index+4])
			numberOfBytes = order.Uint32(data[index+4 : index+8])
			offset = 8
		}
		tmp := data[index+offset:]
		im, err := extractDataElement(&tmp, order, int(dataType), int(numberOfBytes))
		if err != nil {
			return MatMatrix{}, err
		}
		matrix.ImaginaryPart = im
		index += (offset + int(numberOfBytes))
		index = checkIndex(index)
	}
	return matrix, nil
}

func readBytes(m *Matf, numberOfBytes int) ([]byte, error) {
	data := make([]byte, numberOfBytes)
	count, err := m.file.Read(data)
	if err != nil {
		return nil, err
	}

	if count != numberOfBytes {
		return nil, fmt.Errorf("Could not read enough bytes")
	}
	return data, nil
}

func readDataElementField(m *Matf, order binary.ByteOrder) (int, interface{}, error) {
	tag, err := readBytes(m, 8)
	if err != nil {
		return 0, nil, err
	}

	dataType := order.Uint32(tag[:4])
	numberOfBytes := order.Uint32(tag[4:8])

	fmt.Println("DataType: ", dataType, "\tNumberOfBytes: ", numberOfBytes)

	data, err := readBytes(m, int(numberOfBytes))
	if err != nil {
		return 0, nil, err
	}

	switch int(dataType) {
	case MiCompressed:
		return 0, nil, fmt.Errorf("MiCompressed is not yet implemented")
	case MiMatrix:
		extractMatrix(data, order)
	case MiInt8:
		fallthrough
	case MiUint8:
		fallthrough
	case MiInt16:
		fallthrough
	case MiUint16:
		fallthrough
	case MiInt32:
		fallthrough
	case MiUint32:
		fallthrough
	case MiInt64:
		fallthrough
	case MiDouble:
		fallthrough
	case MiUint64:
		extractDataElement(&data, order, int(dataType), int(numberOfBytes))
	default:
		return int(dataType), nil, fmt.Errorf("Data Type %d is not supported", dataType)
	}

	return int(dataType), nil, nil
}

// Open a MAT-file
func Open(file string) (*Matf, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	mat := new(Matf)
	mat.file = f

	err = readHeader(mat, f)
	if err != nil {
		return nil, err
	}

	return mat, nil
}

// ReadDataElement returns the next data element
func ReadDataElement(file *Matf) (int, interface{}, error) {
	if file.byteSwapping == true {
		return readDataElementField(file, binary.LittleEndian)
	}
	return readDataElementField(file, binary.BigEndian)

}

// Close a MAT-file
func Close(file *Matf) error {
	return file.file.Close()
}
