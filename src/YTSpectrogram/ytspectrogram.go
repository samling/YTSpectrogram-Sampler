package main

import "github.com/mdlayher/waveform"
import "io"
import "math"
import "os"
import "fmt"

func GetSampleValues(r io.Reader, options ...waveform.OptionsFunc) ([]float64, error) {
	w, err := waveform.New(r, options...)
	if err != nil {
		return nil, err
	}

	values, err := w.Compute()
	return values, nil
}

// https://gist.github.com/DavidVaini/10308388
func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func main() {
	r, err := os.Open("./audio/test.flac")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	max := 0
	resolution := uint(4)

	values, err := GetSampleValues(r,
		nil,
		nil,
		waveform.Resolution(resolution),
		nil,
		waveform.ScaleClipping(),
		waveform.Sharpness(1),
	)

	for _, f := range values {
		max = int(math.Max(float64(max), Round((f*1E6), .5, 0)))
	}

	for t, f := range values {
		adjusted := Round((f*1E6), .5, 0) / float64(max)
		fmt.Printf("%d,%.2f,%d\n", t, adjusted, int(resolution))
	}
}
