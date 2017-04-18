package main

import "github.com/kennygrant/sanitize"
import "github.com/mdlayher/waveform"
import "encoding/json"
import "fmt"
import "io"
import "io/ioutil"
import "math"
import "os"

// Piggybacking off the waveform library to retrieve just the sample values instead of an image
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
	// Round a float64 {val} if its least significant figure >= {roundOn} to {places} sigfigs
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
	// Protect against path traversal via injection
	filename := sanitize.BaseName(os.Args[1])

	// Open an IO Reader for our FLAC file
	r, err := os.Open("./audio/" + filename + ".flac")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	// Set some basic values
	// max = Tracks the maximum sample value
	// resolution = Specifies the sample rate per second (default 4)
	// m = Map of quarter second to value
	max := 0
	resolution := uint(4)
	m := make(map[int]interface{})

	// Get sample values with our default values
	values, err := GetSampleValues(r,
		nil,
		nil,
		waveform.Resolution(resolution),
		nil,
		waveform.ScaleClipping(),
		waveform.Sharpness(1),
	)

	// Determine the maximum value
	for _, f := range values {
		max = int(math.Max(float64(max), Round((f*1E6), .5, 0)))
	}

	// Adjust our values to be percentages of the max
	for t, f := range values {
		adjusted := Round((f*1E6), .5, 0) / float64(max)
		m[t] = adjusted
	}

	// TODO: Average our values to smooth out any blips and outliers
	// TODO: Identify them first?

	// Serialize our map as a JSON dict
	jsonKeys, _ := json.Marshal(m)

	// Write the results to a json file
	err = ioutil.WriteFile("./out.json", jsonKeys, 0644)
	if err != nil {
		panic(err)
	}

	// Print the map
	fmt.Println(string(jsonKeys))
}
