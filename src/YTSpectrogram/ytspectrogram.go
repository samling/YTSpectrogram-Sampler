package main

import (
	_ "database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jmoiron/sqlx"
	_ "github.com/kennygrant/sanitize"
	"github.com/mdlayher/waveform"
	"io"
	"log"
	"math"
	"os"
)

type Data struct {
	Samples []Sample
}

type Sample struct {
	Timestamp  int
	SampleData float64
}

// Piggybacking off the waveform library to retrieve just the sample values instead of an image
func GetSampleValues(r io.Reader, options ...waveform.OptionsFunc) ([]float64, error) {
	w, err := waveform.New(r, options...)
	if err != nil {
		log.Fatal("Unable to generate waveform data: ", err)
	}

	values, err := w.Compute()
	if err != nil {
		log.Fatal("Unable to compute sample data: ", err)
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
	id := os.Args[1]

	// Open an IO Reader for our FLAC file
	r, err := os.Open("./audio/" + id + ".flac")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	// Initialize some variables
	var (
		max  = 0
		data Data
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

	// Adjust our values to be percentages of the max and add them to our data struct
	for time, val := range sampleValues {
		adjustedVal := Round((val*1E6), .5, 0) / float64(max)
		data.Samples = append(data.Samples, Sample{time, adjustedVal})
	}

	// TODO: Average our values to smooth out any blips and outliers
	// TODO: Identify them first?

	// Serialize our map as a JSON dict
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatal("Cannot encode to JSON: ", err)
	}

	// Connect to our database
	//connString := GetConnectionString(os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))
	//db := sqlx.MustConnect("mysql", connString)

	//// Write id and sampledata to db, ignore duplicates
	//tx := db.MustBegin()
	//tx.MustExec("INSERT INTO Sample (Id, SampleData) VALUES (?, ?) ON DUPLICATE KEY UPDATE Id=Id", id, jsonData)
	//tx.Commit()

	// Print the map
	fmt.Println(string(jsonData))
}
