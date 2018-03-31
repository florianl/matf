package matf

import (
	"fmt"
	"os"
)

type Matf struct {
	Header
}

type Header struct {
	Text                string // 0 - 116
	SubsystemDataOffset []byte // 117 - 124
	Flags               []byte // 125 - 128
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
	mat.Header.Flags = data[124:128]

	return nil
}

func Open(file string) (*Matf, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	mat := new(Matf)

	readHeader(mat, f)

	return mat, nil
}
