# matf [![Build Status](https://travis-ci.org/florianl/matf.svg?branch=master)](https://travis-ci.org/florianl/matf) [![Coverage Status](https://coveralls.io/repos/github/florianl/matf/badge.svg?branch=master)](https://coveralls.io/github/florianl/matf?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/florianl/matf)](https://goreportcard.com/report/github.com/florianl/matf) [![GoDoc](https://godoc.org/github.com/florianl/matf?status.svg)](https://godoc.org/github.com/florianl/matf)

This is `matf` and it is written in [golang](https://golang.org/). `matf` extracts the content from [MATLAB](https://mathworks.com)s MAT-files.

Why?
----
The idea of this project is, to make the content of `mat`-files available in [golang](https://golang.org/).

Example
-------

Load a simple matrix, saved in a [matf](https://mathworks.com)-file, in [gonum/mat](https://godoc.org/gonum.org/v1/gonum/mat).

```golang
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"

	"github.com/florianl/matf"
	"gonum.org/v1/gonum/mat"
)

func main() {

	modelfile, err := matf.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
		return
	}
	defer matf.Close(modelfile)

	element, err := matf.ReadDataElement(modelfile)
	if err != nil && err != io.EOF {
		log.Fatal(err)
		return
	}
	r, c, _, err := element.Dimensions()
	data := []float64{}
	slice := reflect.ValueOf(element.Content.(matf.NumPrt).RealPart)
	for i := 0; i < slice.Len(); i++ {
		value := reflect.ValueOf(slice.Index(i).Interface()).Float()
		data = append(data, value)
	}

	dense := mat.NewDense(r, c, data)
	fmt.Printf("dense = %v\n", mat.Formatted(dense, mat.Prefix("        ")))

}

```

Simple example, using [gorgonia](https://github.com/gorgonia/gorgonia).
```golang
package main

import (
	"io"
	"log"
	"os"
	"reflect"

	"github.com/florianl/matf"
	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"

)

func main() {

	modelfile, err := matf.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
		return
	}
	defer matf.Close(modelfile)

	element, err := matf.ReadDataElement(modelfile)
	if err != nil && err != io.EOF {
		log.Fatal(err)
		return
	}
	r, c, _, err := element.Dimensions()
	data := []float64{}
	slice := reflect.ValueOf(element.Content.(matf.NumPrt).RealPart)
	for i := 0; i < slice.Len(); i++ {
		value := reflect.ValueOf(slice.Index(i).Interface()).Float()
		data = append(data, value)
	}

	t := tensor.New(tensor.WithShape(r,c), tensor.WithBacking(data))

	g := gorgonia.NewGraph()
	w := gorgonia.NewMatrix(g, gorgonia.Float64, gorgonia.WithShape(r,c), gorgonia.WithValue(t))
	gorgonia.Must(gorgonia.Sigmoid(w))

	m := gorgonia.NewTapeMachine(g)
	if err := m.RunAll(); err != nil {
		log.Fatal(err)
		return
	}

	m.Reset()
}
```
