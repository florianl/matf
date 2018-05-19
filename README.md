# matf [![Build Status](https://travis-ci.org/florianl/matf.svg?branch=master)](https://travis-ci.org/florianl/matf) [![Coverage Status](https://coveralls.io/repos/github/florianl/matf/badge.svg?branch=master)](https://coveralls.io/github/florianl/matf?branch=master)

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
