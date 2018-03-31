package matf

import (
	"encoding/binary"
	"fmt"
	"os"
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

// Matf represents the MAT-file
type Matf struct {
	Header
	file         *os.File
	byteSwapping bool
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

func readDataElement(m *Matf, order binary.ByteOrder) (int, interface{}, error) {
	data := make([]byte, 8)
	count, err := m.file.Read(data)
	if err != nil {
		return 0, nil, err
	}

	if count != 8 {
		return 0, nil, fmt.Errorf("Could not read enough bytes")
	}

	dataType := order.Uint32(data[:4])
	numberOfBytes := order.Uint32(data[4:8])

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

	readHeader(mat, f)

	return mat, nil
}

// ReadDataElement returns the next data element
func ReadDataElement(file *Matf) (int, interface{}, error) {
	if file.byteSwapping == true {
		return readDataElement(file, binary.LittleEndian)
	}
	return readDataElement(file, binary.BigEndian)

}

// Close a MAT-file
func Close(file *Matf) error {
	return file.file.Close()
}
