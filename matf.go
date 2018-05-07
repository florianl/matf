package matf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"reflect"
)

// Flags
const (
	ClassMask   = 0x0F
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
	X, Y, Z int
}

// NumPrt contains the numeric part of a matrix
type NumPrt struct {
	RealPart      interface{}
	ImaginaryPart interface{}
}

// StructPrt represents a matf struct
type StructPrt struct {
	FieldNames  []string
	FieldValues []interface{}
}

// CellPrt represents a list of matf cells
type CellPrt struct {
	Cells []MatMatrix
}

// CharPrt represents a matf char array
type CharPrt struct {
	CharName   string
	CharValues []interface{}
}

// MatMatrix represents a matrix
type MatMatrix struct {
	Name  string
	Flags uint32
	Class uint32
	Dimensions
	Content interface{}
}

// Header contains informations about the MAT-file
type Header struct {
	Text                string // 0 - 116
	subsystemDataOffset []byte // 117 - 124
	Version             uint16 // 125 - 126
	endianIndicator     uint16 // 127 - 128
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
	mat.Header.subsystemDataOffset = data[116:124]
	mat.Header.Version = binary.BigEndian.Uint16(data[124:126])
	mat.Header.endianIndicator = binary.BigEndian.Uint16(data[126:128])

	if mat.Header.endianIndicator == binary.BigEndian.Uint16([]byte{0x49, 0x4d}) {
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
			dim.X = int(value)
		case 1:
			dim.Y = int(value)
		case 2:
			dim.Z = int(value)
		default:
			return Dimensions{}, fmt.Errorf("More dimensions than exptected")
		}
	}
	return dim, nil
}

func isSmallDataElementFormat(data *[]byte, order binary.ByteOrder) (bool, error) {
	var offset int

	if order == binary.LittleEndian {
		offset = 2
	}

	small := make([]byte, 2)
	buf := bytes.NewReader((*data)[offset:])
	if err := binary.Read(buf, order, &small); err != nil {
		return false, fmt.Errorf("Could not read bytes: %v", err)
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

func extractMatrix(data []byte, order binary.ByteOrder) (MatMatrix, int, error) {
	var matrix MatMatrix
	var index int
	var offset int
	var dataType uint32
	var numberOfBytes uint32
	var complexNumber bool
	var err error
	var buf *bytes.Reader
	var maxLen = len(data)

	// Array Flags
	_, numberOfBytes, offset, err = extractTag(&data, order)
	if err != nil {
		return MatMatrix{}, 0, err
	}
	arrayFlags := make([]byte, int(numberOfBytes))
	buf = bytes.NewReader(data[index+offset : index+offset+int(numberOfBytes)])
	if err := binary.Read(buf, order, &arrayFlags); err != nil {
		return MatMatrix{}, 0, err
	}
	matrix.Flags = binary.LittleEndian.Uint32(arrayFlags)
	if FlagComplex&matrix.Flags == FlagComplex {
		complexNumber = true
	}
	matrix.Class = matrix.Flags & 0x0F
	index = checkIndex(index + offset + int(numberOfBytes))

	// Dimensions Array
	tmp := data[index : index+8]
	dataType, numberOfBytes, offset, err = extractTag(&tmp, order)
	if err != nil {
		return MatMatrix{}, 0, err
	}
	dimArray := make([]byte, int(numberOfBytes))
	buf = bytes.NewReader(data[index+offset : index+offset+int(numberOfBytes)])
	if err := binary.Read(buf, order, &dimArray); err != nil {
		return MatMatrix{}, 0, err
	}

	dims, _, err := extractDataElement(&dimArray, order, int(dataType), int(numberOfBytes))
	if err != nil {
		return MatMatrix{}, 0, err
	}
	matrix.Dimensions, _ = readDimensions(dims)
	index = checkIndex(index + offset + int(numberOfBytes))

	// Array Name
	tmp = data[index:]
	arrayName, step, err := extractArrayName(&tmp, order)
	matrix.Name = arrayName
	index = checkIndex(index + step)

	switch int(matrix.Class) {
	case MxCellClass:
		var content CellPrt
		for {
			if index >= maxLen {
				break
			}
			tmp := data[index+8:]
			element, step, err := extractMatrix(tmp, order)
			if err != nil {
				return MatMatrix{}, 0, err
			}
			content.Cells = append(content.Cells, element)
			index = checkIndex(index + 8 + step)
		}
		matrix.Content = content
	case MxStructClass:
		var elements []interface{}
		var content StructPrt
		// Field Name Length
		fieldNameLength := order.Uint32(data[index+4 : index+8])
		// Field Names
		numberOfFields := order.Uint32(data[index+12:index+16]) / fieldNameLength
		index = checkIndex(index + 16)
		tmp := data[index : index+int(fieldNameLength*numberOfFields)]
		fieldNames, err := extractFieldNames(&tmp, int(fieldNameLength), int(numberOfFields))
		if err != nil {
			return MatMatrix{}, 0, err
		}
		content.FieldNames = fieldNames
		index = checkIndex(index + (int(numberOfFields) * int(fieldNameLength)))
		// Field Values
		for ; numberOfFields > 0; numberOfFields-- {
			var element interface{}
			tmp := data[index : index+8]
			dataType, numberOfBytes, offset, _ := extractTag(&tmp, order)
			tmp = data[index+offset : index+offset+int(numberOfBytes)]
			element, _, err = extractDataElement(&tmp, order, int(dataType), int(numberOfBytes))
			if err != nil {
				return MatMatrix{}, 0, err
			}
			index = checkIndex(index + offset + int(numberOfBytes))
			elements = append(elements, element)
		}
		content.FieldValues = elements
		matrix.Content = content
	case MxCharClass:
		var content CharPrt
		tmp := data[index : index+8]
		_, numberOfBytes, offset, _ := extractTag(&tmp, order)
		tmp = data[index : index+int(offset)+int(numberOfBytes)]
		name, _, err := extractArrayName(&tmp, order)
		if err != nil {
			return MatMatrix{}, 0, err
		}
		content.CharName = name
		index = checkIndex(index + int(offset) + int(numberOfBytes))
		var counter int
		for {
			if index >= maxLen {
				break
			}
			counter++
			if counter > 3 {
				break
			}
			tmp := data[index : index+8]
			dataType, numberOfBytes, offset, _ := extractTag(&tmp, order)
			tmp = data[index+offset : index+offset+int(numberOfBytes)]
			element, _, err := extractDataElement(&tmp, order, int(dataType), int(numberOfBytes))
			if err != nil {
				return MatMatrix{}, 0, err
			}
			content.CharValues = append(content.CharValues, element)
			index = checkIndex(index + offset + int(numberOfBytes))
		}
		matrix.Content = content
	case MxDoubleClass:
		fallthrough
	case MxSingleClass:
		fallthrough
	case MxInt8Class:
		fallthrough
	case MxUint8Class:
		fallthrough
	case MxInt16Class:
		fallthrough
	case MxUint16Class:
		fallthrough
	case MxInt32Class:
		fallthrough
	case MxUint32Class:
		var content NumPrt
		// Real part
		tmp := data[index:]
		re, used, _ := extractNumeric(&tmp, order)
		content.RealPart = re
		index = checkIndex(index + used)
		// Imaginary part (optional)
		if complexNumber {
			tmp = data[index:]
			im, used, _ := extractNumeric(&tmp, order)
			content.ImaginaryPart = im
			index += used
			index = checkIndex(index)
		}
		matrix.Content = content
	default:
		return MatMatrix{}, 0, fmt.Errorf("This type of class is not supported yet: %d", matrix.Class)
	}
	return matrix, index, nil
}

func readBytes(m *Matf, numberOfBytes int) ([]byte, error) {
	data := make([]byte, numberOfBytes)
	count, err := m.file.Read(data)
	if err != nil {
		return nil, err
	}

	if count != int(numberOfBytes) {
		return nil, fmt.Errorf("Could not read %d bytes", numberOfBytes)
	}
	return data, nil
}

func decompressData(data []byte) ([]byte, error) {
	tmp := bytes.NewReader(data)
	var out bytes.Buffer
	r, err := zlib.NewReader(tmp)
	defer r.Close()
	if err != nil {
		return []byte{}, err
	}
	if r != nil && err == nil {
		io.Copy(&out, r)
	}
	return out.Bytes(), err
}

func readDataElementField(m *Matf, order binary.ByteOrder) (int, []interface{}, error) {
	var elements []interface{}
	var element interface{}
	var data []byte
	var dataType, completeBytes uint32
	var offset, i, step int
	tag, err := readBytes(m, 8)
	if err != nil {
		return 0, nil, err
	}

	dataType = order.Uint32(tag[:4])
	completeBytes = order.Uint32(tag[4:8])

	data, err = readBytes(m, int(completeBytes))
	if err != nil {
		return 0, nil, err
	}

	if dataType == uint32(MiCompressed) {
		plain, err := decompressData(data[:completeBytes])
		if err != nil {
			return 0, nil, err
		}
		dataType, completeBytes, offset, err = extractTag(&plain, order)
		data = plain[offset:]
	}

	element, i, err = extractDataElement(&data, order, int(dataType), int(completeBytes))
	if err != nil {
		return 0, nil, err
	}
	elements = append(elements, element)

	for uint32(i) < completeBytes {
		tmp := data[i:]
		dataType, numberOfBytes, offset, err := extractTag(&tmp, order)
		tmp = data[i+offset:]
		element, step, err = extractDataElement(&tmp, order, int(dataType), int(numberOfBytes))
		if err != nil {
			return 0, nil, err
		}
		i = checkIndex(i + step + offset)
		elements = append(elements, element)
	}

	return int(dataType), elements, nil
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
