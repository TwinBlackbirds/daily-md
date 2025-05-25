package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type payload struct {
	UFC event `json:"UFC"`
}

type event struct {
	Number    string `json:"number"`
	Title     string `json:"title"`
	Venue     string `json:"venue"`
	Date      string `json:"date"`
	Timestamp string `json:"timestamp"`
}

func getRedditSummary(c *colly.Collector) string {
	// already written in another app
	c.OnHTML("", func(e *colly.HTMLElement) {

	})
	c.Visit("")
	return ""
}

func getDailyQuote(c *colly.Collector) string {
	c.OnHTML("", func(e *colly.HTMLElement) {

	})
	c.Visit("")
	return ""
}

// for when UFC event has passed, is out of date
func checkIfNeedGetUFC(ufc event) (bool, error) {
	// read json settings file
	_, err := readSettingsFile("settings.json")
	if err != nil {
		return true, err
	}
	// convert timestamp to int
	tsNum, err := strconv.Atoi(ufc.Timestamp)
	if err != nil {
		return true, nil // timestamp does not exist
	}
	// compare timestamps
	currentTime := int(time.Now().Unix())
	if currentTime < tsNum {
		return false, nil
	}
	return true, nil
}

// all encompassing function which logically updates JSON
func updateUFCDetails(ufc event) error {
	// see if it is necessary
	existing, err := readSettingsFile("settings.json")
	need, err := checkIfNeedGetUFC(ufc)
	if err != nil {
		return err
	}
	// return early if not
	if !need && existing.UFC.Number == ufc.Number {
		fmt.Println("Not updating UFC event details as event has not passed!")
		return nil
	}
	fmt.Println("Updating UFC event details")
	ptr := payload{
		UFC: ufc,
	}
	err = writeSettingsFile("settings.json", ptr)
	if err != nil {
		return err
	}
	return nil
}

// read settings file to byte slice
func getSettingsFileBytes(filename string) ([]byte, error) {
	// open json settings file
	file, err := os.Open(filename)
	if err != nil {
		file, err = os.Create(filename)
		if err != nil {
			return nil, err
		}
	}
	// defer file closure
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(file)

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// wipe settings
func clearSettingsFile(filename string) error {
	err := os.Remove(filename)
	err = os.WriteFile(filename, nil, 0644)
	if err != nil {
		return err
	}
	return nil
}

// write settings payload
func writeSettingsFile(filename string, settings payload) error {
	// clear existing file
	err := clearSettingsFile(filename)
	// create json and write settings file
	jsonData, err := json.Marshal(settings)
	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return err
	}
	return nil
}

// read settings payload
func readSettingsFile(filename string) (payload, error) {
	// convenience wrapper
	//fmt.Println("Reading settings file to bytes..")
	byteValue, err := getSettingsFileBytes(filename)
	if err != nil {
		return payload{}, err
	}
	//fmt.Println("Converting bytes to payload..")
	return convertBytesToPayload(byteValue)
}

// byte slice -> settings payload
func convertBytesToPayload(bytes []byte) (payload, error) {
	var ptr payload
	err := json.Unmarshal(bytes, &ptr)
	if err != nil {
		return payload{}, nil
	}
	return ptr, nil
}

// scraper: get ufc event details in a logical manner
func getUFCDetails(c *colly.Collector) (event, error) {
	fmt.Println("Reading saved UFC event details..")

	// get ufc event number (find our place)
	ptr, err := readSettingsFile("settings.json")
	if err != nil {
		return event{}, err
	}

	// validate existing ufc number
	fmt.Println("Checking if the details are current..")
	need, err := checkIfNeedGetUFC(ptr.UFC)
	if err != nil {
		return event{}, err
	}
	if !need {
		fmt.Println("Details are already current!")
		return ptr.UFC, nil
	}

	// scraper portion
	fmt.Println("Details are not current, finding current details!")
	if ptr.UFC.Number == "" {
		ptr.UFC.Number = "316" //fallback
	}
	url := "https://www.ufc.com/event/ufc-" + ptr.UFC.Number
	fmt.Printf("Using url: %s\n", url)

	// setup capture variables
	title := "None"
	dateAndTime := "None"
	venue := "None"
	timestamp := "None"

	// DOM scraping setup
	c.OnHTML("div[class*='node--type-event']", func(e *colly.HTMLElement) {
		title = e.ChildText("div[class*='headline'] > .e-divider")
		dateAndTimeSel := "div[class*='suffix'][data-locale='en-can']"
		dateAndTime = e.ChildText(dateAndTimeSel)
		venue = e.ChildText("div[class*='hero__text']")
		timestamp = e.ChildAttr(dateAndTimeSel, "data-timestamp")
	})

	err = c.Visit(url) // perform scraping
	if err != nil {
		return event{}, err
	}

	// clean up results
	title = sanitizeTitle(title)
	dateAndTime = sanitize(dateAndTime)
	venue = sanitize(venue)
	timestamp = sanitize(timestamp)
	UFCEvent := event{
		Title:     title,
		Number:    ptr.UFC.Number,
		Venue:     venue,
		Date:      dateAndTime,
		Timestamp: timestamp,
	}
	// update settings.json
	defer func() {
		err = updateUFCDetails(UFCEvent) // only updates it if necessary
		if err != nil {
			log.Fatal(err)
		}
	}()

	return UFCEvent, nil
}

// --- weather types ---

type Seconds int64
type Latitude float64
type Longitude float64
type Milliseconds float64
type Elevation float64

type Date string
type Hour string

func (h Hour) String() string {
	suffix := "AM"
	split := strings.Split(string(h), ":")
	hr, err := strconv.Atoi(split[0])
	if err != nil {
		log.Fatal(err)
	}
	m := split[1]
	if hr == 0 { // midnight
		hr = 12
	}
	if hr > 12 {
		hr = hr - 12
		suffix = "PM"
	}
	return fmt.Sprintf("%02d:%s %s", hr, m, suffix)
}

type Time string

func (t Time) String() string {
	split := strings.Split(string(t), "T")
	date := Date(split[0])
	hour := Hour(split[1])
	return fmt.Sprintf("%s at %s", date, hour)
}

type Temperature float64

func (t Temperature) String() string {
	conv := strconv.FormatFloat(float64(t), 'f', 2, 64)
	return conv
}

// HourlyUnits contains strings describing the units used in Hourly
type HourlyUnits struct {
	Time                   string `json:"time"`
	TemperatureAtTwoMetres string `json:"temperature_2m"`
}
type Hourly struct {
	Time                   []Time        `json:"time"`
	TemperatureAtTwoMetres []Temperature `json:"temperature_2m"`
}

type HourlyWithUnits struct {
	Units  HourlyUnits
	Hourly Hourly
}

func (h *HourlyWithUnits) get(pos uint) (Time, Temperature) {
	return h.Hourly.Time[pos], h.Hourly.TemperatureAtTwoMetres[pos]
}
func (h *HourlyWithUnits) printAll() error {
	if len(h.Hourly.TemperatureAtTwoMetres) != len(h.Hourly.Time) {
		return errors.New("hourly temperature and temperature are not the same length")
	}
	for i := 0; i < len(h.Hourly.Time); i++ {
		_time, temp := h.get(uint(i))
		fmt.Printf("%s, %.1f%s\n", _time, temp, h.Units.TemperatureAtTwoMetres)
	}
	return nil
}

// getDay returns a new Hourly which contains the n*24 - n+1*24 elements of h
func (h *HourlyWithUnits) getDay(day uint) (Day, error) {
	if day < 0 || day > 4 {
		return Day{}, errors.New("day must be between 0 and 4")
	}
	times := [24]Time(h.Hourly.Time[day*24 : (day+1)*24])
	temps := [24]Temperature(h.Hourly.TemperatureAtTwoMetres[day*24 : (day+1)*24])
	newDay := Day{
		Units: h.Units,
		Times: times,
		Temps: temps,
	}
	return newDay, nil
}

type Day struct {
	Units HourlyUnits
	Times [24]Time // is 24 long always
	Temps [24]Temperature
}

func (d *Day) get(hour uint) (Time, Temperature, error) {
	if hour > 23 {
		return "", 0, errors.New("hour is out of range")
	}
	return d.Times[hour], d.Temps[hour], nil
}
func (d *Day) printAll() {
	for i := 0; i < len(d.Times); i++ {
		_time, temp, _ := d.get(uint(i)) // we can safely discard the err as we are within bounds
		fmt.Printf("%s, %.1f%s\n", _time, temp, d.Units.TemperatureAtTwoMetres)
	}
}

type WeatherResponse struct {
	Latitude       Latitude     `json:"latitude"`
	Longitude      Longitude    `json:"longitude"`
	GenerationTime Milliseconds `json:"generation_time_ms"`
	UtcOffset      Seconds      `json:"utc_offset_seconds"` // can be negative
	Timezone       string       `json:"timezone"`
	TimezoneAbbrev string       `json:"timezone_abbreviation"`
	Elevation      Elevation    `json:"elevation"`
	HourlyUnits    HourlyUnits  `json:"hourly_units"`
	Hourly         Hourly       `json:"hourly"`
}

type WeatherRequest struct {
	Latitude  Latitude  `json:"latitude"`
	Longitude Longitude `json:"longitude"`
}

func (w *WeatherRequest) getWeather() (WeatherResponse, error) {

	// construct request
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&hourly=temperature_2m&timezone=auto",
		w.Longitude, w.Latitude)

	// get weather for long, lat
	resp, err := http.Get(url)
	if err != nil {
		return WeatherResponse{}, err
	}
	// close body -- what does this mean
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(resp.Body)

	// bad request
	if resp.StatusCode != 200 {
		return WeatherResponse{}, errors.New(resp.Status)
	}

	var weather WeatherResponse
	err = json.NewDecoder(resp.Body).Decode(&weather)
	if err != nil {
		return WeatherResponse{}, err
	}

	return weather, nil
}

type FormattedResponse struct {
	Latitude        Latitude
	Longitude       Longitude
	GenerationTime  Milliseconds
	UtcOffset       Seconds // can be negative
	Timezone        string
	TimezoneAbbrev  string
	Elevation       Elevation
	HourlyWithUnits HourlyWithUnits
}

func (w *WeatherResponse) formatResponse() FormattedResponse {
	hwu := HourlyWithUnits{
		Hourly: w.Hourly,
		Units:  w.HourlyUnits,
	}
	f := FormattedResponse{
		Latitude:        w.Latitude,
		Longitude:       w.Longitude,
		GenerationTime:  w.GenerationTime,
		UtcOffset:       w.UtcOffset,
		Timezone:        w.Timezone,
		TimezoneAbbrev:  w.TimezoneAbbrev,
		Elevation:       w.Elevation,
		HourlyWithUnits: hwu,
	}
	return f
}

func sanitize(s string) string {
	clone := strings.Clone(s)
	clone = strings.Replace(clone, "   ", " ", -1) // 3 -> 1
	clone = strings.Replace(clone, "  ", "", -1)   // 2 -> 0
	clone = strings.Replace(clone, "\n", "", -1)   // \n ->
	return clone
}

func sanitizeTitle(t string) string {
	clone := strings.Clone(t)
	clone = strings.Replace(clone, "\n", " ", -1)
	clone = strings.Replace(clone, "    ", " ", -1)
	clone = strings.Replace(clone, "   ", " ", -1)
	clone = strings.Replace(clone, "  ", " ", -1)
	return clone
}

func main() {
	// get time
	currentTime := time.Now()
	year, month, day := currentTime.Date()
	// create output folder
	fmt.Println("Checking if 'markdown' directory exists..")
	folder := "markdown"
	_, err := os.ReadDir(folder)
	if err != nil {
		err := os.Mkdir(folder, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	// check if previous file exists (and construct filename)
	fmt.Println("Checking if today's markdown file already exists..")
	filename := fmt.Sprintf("%s/%d-%02d-%02d.md", folder, year, month, day)
	// remove previous file
	_, err = os.ReadFile(filename)
	if err == nil {
		err = os.Remove(filename)
		if err != nil {
			log.Fatal(err)
		}
	}
	// create file
	fmt.Println("Creating today's markdown file..")
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	// (defer) close file ptr
	defer func(file *os.File) {
		fmt.Println("Closing file..")
		err := file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(file)

	//// make new collector
	//fmt.Println("Initiating collector to use across all bots..")
	//c := colly.NewCollector()

	// ufc
	//fmt.Println("Attempting to retrieve UFC details..")
	//ufc, err := getUFCDetails(c)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//print("Found UFC event: ")
	//format := fmt.Sprintf("UFC %s - %s | %s | %s", ufc.Number, ufc.Title, ufc.Date, ufc.Venue)
	//fmt.Println(format)

	// get weather
	fmt.Println("Attempting to get weather details..")
	// load .env file
	err = godotenv.Load(".env")
	// get long & lat coordinates from .env file
	long, err := strconv.ParseFloat(os.Getenv("DAILY_MD_LONG"), 64)
	lat, err := strconv.ParseFloat(os.Getenv("DAILY_MD_LAT"), 64)
	req := WeatherRequest{
		Longitude: Longitude(long),
		Latitude:  Latitude(lat),
	}
	res, err := req.getWeather()
	if err != nil {
		log.Fatal(err)
	}
	formattedResponse := res.formatResponse()
	// today's weather
	today, err := formattedResponse.HourlyWithUnits.getDay(0)
	if err != nil {
		log.Fatal(err)
	}
	midnight, tempAtMidnight, _ := today.get(0) // we can discard the err as we are within bounds
	fmt.Printf("%s - %s%s\n", midnight, tempAtMidnight, today.Units.TemperatureAtTwoMetres)
	fmt.Println("All commands executed!")
}
