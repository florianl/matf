package matf

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
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
	Chars []string
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
		return errors.Wrap(err, "\nfile.Read() in readHeader() failed")
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

func alignIndex(r io.Reader, order binary.ByteOrder, index int) int {
	for {
		if index%8 == 0 {
			return index
		}
		readMatfBytes(r, order, 1)
		index++
	}
}

func extractClass(mat *MatMatrix, r io.Reader, order binary.ByteOrder) (int, error) {
	var index int

	switch int(mat.Class) {
	case MxCellClass:
		var content CellPrt
		var maxElements = mat.Dim.Y
		var noElements int
		for {
			if noElements >= maxElements {
				break
			}
			// Alignment
			readMatfBytes(r, order, 8)
			element, step, err := extractMatrix(r, order)
			if err != nil {
				return 0, err
			}
			content.Cells = append(content.Cells, element)
			index = alignIndex(r, order, index+8+step)
			noElements++
		}
		mat.Content = content
	case MxStructClass:
		var elements = make(map[string][]interface{})
		var content StructPrt
		data, err := readMatfBytes(r, order, 16)
		if err != nil {
			return 0, fmt.Errorf("Unable to read %d bytes: %v", 16, err)
		}
		// Field Name Length
		fieldNameLength := order.Uint32(data[index+4 : index+8])
		// Field Names
		numberOfFields := order.Uint32(data[index+12:index+16]) / fieldNameLength
		index = alignIndex(r, order, index+16)
		fieldNames, err := extractFieldNames(r, order, int(fieldNameLength), int(numberOfFields))
		if err != nil {
			return 0, err
		}
		content.FieldNames = fieldNames
		index = alignIndex(r, order, index+(int(numberOfFields)*int(fieldNameLength)))
		// Field Values
		toExtract := (mat.Dim.Y * int(numberOfFields))
		var i int
		for ; toExtract > 0; toExtract-- {
			var element interface{}
			dataType, numberOfBytes, offset, _ := extractTag(r, order)
			element, _, err = extractDataElement(r, order, int(dataType), int(numberOfBytes))
			if err != nil {
				return 0, err
			}
			index = alignIndex(r, order, index+offset+int(numberOfBytes))
			elements[fieldNames[i]] = append(elements[fieldNames[i]], element)
			i = (i + 1) % int(numberOfFields)
		}
		content.FieldValues = elements
		mat.Content = content
	case MxCharClass:
		var content CharPrt
		var maxElements = mat.Dim.X

		element, numberOfBytes, err := extractArrayName(r, order)
		if err != nil {
			return 0, err
		}
		index = alignIndex(r, order, index+numberOfBytes+8)
		if maxElements > 1 {
			// Split the elements
			elements := make(map[int][]byte)
			var mapIndex int
			for i, v := range element {
				if i%2 != 0 {
					continue
				}
				elements[mapIndex%maxElements] = append(elements[mapIndex%maxElements], byte(v))
				mapIndex++
			}
			for _, v := range elements {
				content.Chars = append(content.Chars, string(v))
			}
		} else {
			content.Chars = append(content.Chars, element)
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
		re, used, _ := extractNumeric(r, order)
		content.RealPart = re
		index = alignIndex(r, order, index+used)
		// Imaginary part (optional)
		if FlagComplex&mat.Flags == FlagComplex {
			im, used, _ := extractNumeric(r, order)
			content.ImaginaryPart = im
			index += used
			index = alignIndex(r, order, index)
		}
		mat.Content = content
	default:
		return 0, fmt.Errorf("This type of class is not supported yet: %d", mat.Class)
	}

	return index, nil
}

func extractMatrix(r io.Reader, order binary.ByteOrder) (MatMatrix, int, error) {
	var matrix MatMatrix
	var index int
	var offset int
	var dataType uint32
	var numberOfBytes uint32
	var err error

	// Array Flags
	_, numberOfBytes, offset, err = extractTag(r, order)
	if err != nil {
		return MatMatrix{}, 0, errors.Wrap(err, "\nextractTag() in extractMatrix() failed:")
	}
	index = alignIndex(r, order, index+offset+int(numberOfBytes))

	arrayFlags, err := readMatfBytes(r, order, int(numberOfBytes))
	if err != nil {
		return MatMatrix{}, 0, errors.Wrap(err, "\nreadMatfBytes() in extractMatrix() failed:")
	}
	matrix.Flags = order.Uint32(arrayFlags)
	matrix.Class = matrix.Flags & 0x0F
	index = alignIndex(r, order, index+offset+int(numberOfBytes))

	// Dimensions Array
	dataType, numberOfBytes, offset, err = extractTag(r, order)
	if err != nil {
		return MatMatrix{}, 0, errors.Wrap(err, "\nextractTag() in extractMatrix() failed:")
	}
	dims, _, err := extractDataElement(r, order, int(dataType), int(numberOfBytes))
	if err != nil {
		return MatMatrix{}, 0, errors.Wrap(err, "\nextractDataElement() in extractMatrix() failed:")
	}
	matrix.Dim, _ = readDimensions(dims)
	index = alignIndex(r, order, index+offset+int(numberOfBytes))

	// Array Name
	arrayName, step, err := extractArrayName(r, order)
	if err != nil {
		return MatMatrix{}, 0, errors.Wrap(err, "\nextractArrayName() in extractMatrix() failed:")
	}
	matrix.Name = arrayName
	index = alignIndex(r, order, index+step)

	steps, err := extractClass(&matrix, r, order)
	if err != nil {
		return MatMatrix{}, 0, errors.Wrap(err, "\nextractClass() in extractMatrix() failed:")
	}
	index = alignIndex(r, order, index+steps)

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
		return []byte{}, errors.Wrap(err, "\nzlib.NewReader() in decompressData() failed")
	}
	defer r.Close()
	if r != nil {
		io.Copy(&out, r)
	}
	return out.Bytes(), err
}

func readDataElementField(m *Matf, order binary.ByteOrder) (MatMatrix, error) {
	var mat MatMatrix
	var data []byte
	var dataType, completeBytes uint32
	tag, err := readBytes(m, 8)
	if err != nil {
		return MatMatrix{}, err
	}

	dataType = order.Uint32(tag[:4])
	completeBytes = order.Uint32(tag[4:8])
	data, err = readBytes(m, int(completeBytes))
	if err != nil {
		return MatMatrix{}, errors.Wrap(err, "\nreadBytes() in readDataElementField() failed")
	}

	if dataType == uint32(MiCompressed) {
		plain, err := decompressData(data[:completeBytes])
		if err != nil {
			return MatMatrix{}, errors.Wrap(err, "\ndecompressData() in readDataElementField() failed")
		}
		dataType = order.Uint32(plain[:4])
		completeBytes = order.Uint32(plain[4:8])
		if err != nil {
			return MatMatrix{}, errors.Wrap(err, "\nextractTag() in readDataElementField() failed")
		}
		data = plain[8:]
	}

	tmpfile, err := ioutil.TempFile("", "matf")
	if err != nil {
		return MatMatrix{}, errors.Wrap(err, "\nioutil.TempFile() in readDataElementField() failed")
	}

	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	}()

	if _, err = tmpfile.Write(data); err != nil {
		return MatMatrix{}, errors.Wrap(err, "\nos.Write() in readDataElementField() failed")
	}
	tmpfile.Seek(0, 0)
	r := bufio.NewReader(tmpfile)

	element, i, err := extractDataElement(r, order, int(dataType), int(completeBytes))
	if err != nil {
		return MatMatrix{}, errors.Wrap(err, "\nextractDataElement() in readDataElementField() failed")
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
		return nil, errors.Wrap(err, "\nreadHeader() in Open() failed")
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
