package main

import (
	_ "database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/kennygrant/sanitize"
	"github.com/mdlayher/waveform"
	"io"
	"math"
	"os"
)

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

func GetConnectionString(dbHost string, dbUser string, dbPass string, dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", dbUser, dbPass, dbHost, dbName)
}

func main() {
	// Protect against path traversal via injection
	// TODO: Resolve conflicts with URLs that have "_" in them
	//hash := sanitize.Path(os.Args[1])
	hash := os.Args[1]

	// Open an IO Reader for our FLAC file
	r, err := os.Open("./audio/" + hash + ".flac")
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
	jsonData, _ := json.Marshal(m)

	// Connect to our database
	connString := GetConnectionString(os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))
	db := sqlx.MustConnect("mysql", connString)

	// Write hash and sampledata to db, ignore duplicates
	tx := db.MustBegin()
	tx.MustExec("INSERT INTO Sample (Hash, SampleData) VALUES (?, ?) ON DUPLICATE KEY UPDATE hash=hash", hash, jsonData)
	tx.Commit()

	// Print the map
	//fmt.Println(string(jsonData))
}
