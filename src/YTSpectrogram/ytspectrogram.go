package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mdlayher/waveform"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
)

type Data struct {
	Id         string
	SampleData []Sample
}

type Sample struct {
	Timestamp int
	Value     float64
}

// Piggybacking off the waveform library to retrieve just the sample values instead of an image
func GetSampleValues(r io.Reader, options ...waveform.OptionsFunc) ([]float64, error) {
	w, err := waveform.New(r, options...)
	if err != nil {
		log.Println("Unable to generate waveform data: ", err)
	}

	values, err := w.Compute()
	if err != nil {
		log.Println("Unable to compute sample data: ", err)
	}

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

// Formulate a connection string from environment variables
func GetConnectionString(dbHost string, dbUser string, dbPass string, dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", dbUser, dbPass, dbHost, dbName)
}

func main() {
	// Get the ID of our video from the docker run argument
	id := os.Getenv("YTID")

	// Open an IO Reader for our FLAC file
	r, err := os.Open("./audio/" + id + ".flac")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	// Initialize some variables
	var (
		max    = 0
		data   Data
		apiUrl = "http://sboynton.com:3001/api/Samples"
	)

	// Get sample data with our default values
	sampleValues, err := GetSampleValues(r,
		nil,
		nil,
		waveform.Resolution(4),
		nil,
		waveform.ScaleClipping(),
		waveform.Sharpness(1),
	)

	// Determine the maximum amplitude value
	for _, val := range sampleValues {
		max = int(math.Max(float64(max), Round((val*1E6), .5, 0)))
	}

	// Build a data struct for our audio
	data.Id = id

	// Adjust our values to be percentages of the max and add them to our data struct
	for time, val := range sampleValues {
		adjustedVal := Round((val*1E6), .5, 0) / float64(max)
		data.SampleData = append(data.SampleData, Sample{time, adjustedVal})
	}

	// TODO: Average our values to smooth out any blips and outliers
	// TODO: Identify them first?

	// Serialize our map as a JSON dict
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println("Cannot encode to JSON: ", err)
	}

	// Write our JSON data to a byte array for POSTing
	jsonStr := []byte(jsonData)

	// POST our data to our API endpoint
	client := http.Client{}
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Unable to reach the server.")
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
	}
}
