/*

Package matf extracts the content from MAT-files and make it available in
golang.  In golang, then you can use your favorite Machine Learning environment,
to further use of the extracted data.

For example, you can use the data in gonum:
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

Or use it in gorgonia:

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
*/
package matf
