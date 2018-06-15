package matf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/pkg/errors"
)

// Basic binary flags for various array types
const (
	ClassMask   = 0x0F    // Mask to extract the containing class from an array.
	FlagComplex = 1 << 11 // If set, the data element contains an imaginary part.
	FlagGlobal  = 1 << 10 // MATLAB uses this element on global scope.
	FlagLogical = 1 << 9  // Array is used for logical indexing.
)

// Matf represents the MAT-file
type Matf struct {
	Header
	file         *os.File
	byteSwapping bool
}

// Dim contains the dimensions of a MatMatrix
type Dim struct {
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
	FieldValues map[string][]interface{}
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
	Dim
	Content interface{} // Can contain NumPrt, StructPrt, CellPrt or CharPrt - depending on the value in Class.
}

// Header contains informations about the MAT-file
type Header struct {
	Text                string // Some kind of descriptive text containing various information.
	SubsystemDataOffset []byte // Contains the sybsystem-specific data.
	Version             uint16 // MATLAB version used, to create this file.
	EndianIndicator     uint16 // Indicates, if the file was written on a Big Endian or Little Endian system.
}

func readHeader(mat *Matf, file *os.File) error {
	data := make([]byte, 128)
	count, err := file.Read(data)
	if err != nil {
		return errors.Wrap(err, "file.Read() in readHeader() failed")
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

func readDimensions(data interface{}) (Dim, error) {
	var dim Dim
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
			return Dim{}, fmt.Errorf("More dimensions than exptected")
		}
	}
	return dim, nil
}

func checkIndex(index int) int {
	for {
		if index%8 == 0 {
			return index
		}
		index++
	}
}

func extractClass(mat *MatMatrix, maxIndex int, data *[]byte, order binary.ByteOrder) (int, error) {
	var index int

	switch int(mat.Class) {
	case MxCellClass:
		var content CellPrt
		for {
			if index >= maxIndex {
				break
			}
			tmp := (*data)[index+8:]
			element, step, err := extractMatrix(tmp, order)
			if err != nil {
				return 0, err
			}
			content.Cells = append(content.Cells, element)
			index = checkIndex(index + 8 + step)
		}
		mat.Content = content
	case MxStructClass:
		var elements = make(map[string][]interface{})
		var content StructPrt
		// Field Name Length
		fieldNameLength := order.Uint32((*data)[index+4 : index+8])
		// Field Names
		numberOfFields := order.Uint32((*data)[index+12:index+16]) / fieldNameLength
		index = checkIndex(index + 16)
		tmp := (*data)[index : index+int(fieldNameLength*numberOfFields)]
		fieldNames, err := extractFieldNames(&tmp, int(fieldNameLength), int(numberOfFields))
		if err != nil {
			return 0, err
		}
		content.FieldNames = fieldNames
		index = checkIndex(index + (int(numberOfFields) * int(fieldNameLength)))
		// Field Values
		toExtract := (mat.Dim.Y * int(numberOfFields))
		var i int
		for ; toExtract > 0; toExtract-- {
			var element interface{}
			tmp := (*data)[index : index+8]
			dataType, numberOfBytes, offset, _ := extractTag(&tmp, order)
			tmp = (*data)[index+offset : index+offset+int(numberOfBytes)]
			element, _, err = extractDataElement(&tmp, order, int(dataType), int(numberOfBytes))
			if err != nil {
				return 0, err
			}
			index = checkIndex(index + offset + int(numberOfBytes))
			elements[fieldNames[i]] = append(elements[fieldNames[i]], element)
			i = (i + 1) % int(numberOfFields)
		}
		content.FieldValues = elements
		mat.Content = content
	case MxCharClass:
		var content CharPrt
		tmp := (*data)[index : index+8]
		_, numberOfBytes, offset, _ := extractTag(&tmp, order)
		tmp = (*data)[index : index+int(offset)+int(numberOfBytes)]
		name, _, err := extractArrayName(&tmp, order)
		if err != nil {
			return 0, err
		}
		content.CharName = name
		index = checkIndex(index + int(offset) + int(numberOfBytes))
		for {
			if index >= maxIndex {
				break
			}
			tmp := (*data)[index : index+8]
			dataType, numberOfBytes, offset, _ := extractTag(&tmp, order)
			tmp = (*data)[index+offset : index+offset+int(numberOfBytes)]
			element, _, err := extractDataElement(&tmp, order, int(dataType), int(numberOfBytes))
			if err != nil {
				return 0, err
			}
			content.CharValues = append(content.CharValues, element)
			index = checkIndex(index + offset + int(numberOfBytes))
		}
		mat.Content = content
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
		tmp := (*data)[index:]
		re, used, _ := extractNumeric(&tmp, order)
		content.RealPart = re
		index = checkIndex(index + used)
		// Imaginary part (optional)
		if FlagComplex&mat.Flags == FlagComplex {
			tmp = (*data)[index:]
			im, used, _ := extractNumeric(&tmp, order)
			content.ImaginaryPart = im
			index += used
			index = checkIndex(index)
		}
		mat.Content = content
	default:
		return 0, fmt.Errorf("This type of class is not supported yet: %d", mat.Class)
	}

	return index, nil
}

func extractMatrix(data []byte, order binary.ByteOrder) (MatMatrix, int, error) {
	var matrix MatMatrix
	var index int
	var offset int
	var dataType uint32
	var numberOfBytes uint32
	var err error
	var buf *bytes.Reader
	var maxLen = len(data)

	// Array Flags
	_, numberOfBytes, offset, err = extractTag(&data, order)
	if err != nil {
		return MatMatrix{}, 0, err
	}
	if numberOfBytes != 8 {
		// The size of the Array Flags subelement is always 2 * miUINT32
		return MatMatrix{}, 0, fmt.Errorf("Expected Array Flags field lengt of 8 got %d", numberOfBytes)
	}
	arrayFlags := make([]byte, int(numberOfBytes))
	buf = bytes.NewReader(data[index+offset : index+offset+int(numberOfBytes)])
	if err := binary.Read(buf, order, &arrayFlags); err != nil {
		return MatMatrix{}, 0, err
	}
	matrix.Flags = order.Uint32(arrayFlags)
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
	matrix.Dim, _ = readDimensions(dims)
	index = checkIndex(index + offset + int(numberOfBytes))

	// Array Name
	tmp = data[index:]
	arrayName, step, err := extractArrayName(&tmp, order)
	if err != nil {
		return MatMatrix{}, 0, err
	}
	matrix.Name = arrayName
	index = checkIndex(index + step)

	tmp = data[index:]
	steps, err := extractClass(&matrix, maxLen-index, &tmp, order)
	if err != nil {
		return MatMatrix{}, 0, err
	}
	index = checkIndex(index + steps)

	return matrix, index, nil
}

func readBytes(m *Matf, numberOfBytes int) ([]byte, error) {
	data := make([]byte, numberOfBytes)
	var index int

	for {
		count, err := m.file.Read(data[index:])
		if err != nil {
			return nil, err
		}
		index += count
		if index >= numberOfBytes || count == 0 {
			break
		}
	}
	if index != numberOfBytes {
		return nil, fmt.Errorf("Read %d of %d bytes", index, numberOfBytes)
	}
	return data, nil
}

func decompressData(data []byte) ([]byte, error) {
	tmp := bytes.NewReader(data)
	var out bytes.Buffer
	r, err := zlib.NewReader(tmp)
	if err != nil {
		return []byte{}, errors.Wrap(err, "zlib.NewReader() in decompressData() failed")
	}
	defer r.Close()
	if r != nil {
		io.Copy(&out, r)
	}
	return out.Bytes(), err
}

func readDataElementField(m *Matf, order binary.ByteOrder) (MatMatrix, error) {
	var mat MatMatrix
	var element interface{}
	var data []byte
	var dataType, completeBytes uint32
	var offset, i int
	tag, err := readBytes(m, 8)
	if err != nil {
		return MatMatrix{}, err
	}

	dataType = order.Uint32(tag[:4])
	completeBytes = order.Uint32(tag[4:8])
	data, err = readBytes(m, int(completeBytes))
	if err != nil {
		return MatMatrix{}, errors.Wrap(err, "readBytes() in readDataElementField() failed")
	}

	if dataType == uint32(MiCompressed) {
		plain, err := decompressData(data[:completeBytes])
		if err != nil {
			return MatMatrix{}, errors.Wrap(err, "decompressData() in readDataElementField() failed")
		}
		dataType, completeBytes, offset, err = extractTag(&plain, order)
		if err != nil {
			return MatMatrix{}, errors.Wrap(err, "extractTag() in readDataElementField() failed")
		}
		data = plain[offset:]
	}

	element, i, err = extractDataElement(&data, order, int(dataType), int(completeBytes))
	if err != nil {
		return MatMatrix{}, errors.Wrap(err, "extractDataElement() in readDataElementField() failed")
	}
	if int(dataType) == MiMatrix {
		mat = element.(MatMatrix)
	}

	for uint32(i) < completeBytes {
		return mat, fmt.Errorf("readDataElementField() could not extract all information")
	}

	return mat, nil
}

// Dimensions returns the dimensions of a matrix
func (m MatMatrix) Dimensions() (int, int, int, error) {
	return m.Dim.X, m.Dim.Y, m.Dim.Z, nil
}

// Open a MAT-file and extracts the header information into the Header struct.
func Open(file string) (*Matf, error) {
	if info, err := os.Stat(file); err == nil && info.IsDir() {
		return nil, fmt.Errorf("%s is not a file", file)
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	mat := new(Matf)
	mat.file = f

	err = readHeader(mat, f)
	if err != nil {
		return nil, errors.Wrap(err, "readHeader() in Open() failed")
	}

	return mat, nil
}

// ReadDataElement returns the next data element.
// It returns io.EOF, if no further elements are available
func ReadDataElement(file *Matf) (MatMatrix, error) {
	if file.byteSwapping {
		return readDataElementField(file, binary.LittleEndian)
	}
	return readDataElementField(file, binary.BigEndian)

}

// Close a MAT-file
func Close(file *Matf) error {
	return file.file.Close()
}
