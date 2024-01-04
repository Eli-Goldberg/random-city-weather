package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/hectormalot/omgo"
)

type Coordinates struct {
	Latitude  float64 `json:"lat,string"`
	Longitude float64 `json:"lon,string"`
}

type Country struct {
	Name struct {
		Common string `json:"common"`
	} `json:"name"`
	Capital []string `json:"capital"`
}

func main() {
	c, _ := omgo.NewClient()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Println("Loading random capitals...")
	capitals, err := loadRandomCountriesAndCapitals()
	if err != nil {
		fmt.Printf("Error loading cities: %v", err)
		os.Exit(1)
	}
	fetchWeather(ctx, c, capitals)
}

func loadRandomCountriesAndCapitals() ([]Country, error) {
	// Make an HTTP GET request to the API
	resp, err := http.Get("https://restcountries.com/v3.1/all")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %v", resp.StatusCode)
	}

	// Decode the JSON response into a slice of Country objects
	var countries []Country
	err = json.NewDecoder(resp.Body).Decode(&countries)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}
	return countries, nil
}

func fetchWeather(ctx context.Context, client omgo.Client, capitals []Country) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			city := getRandomCity(capitals)
			coordinates, err := GetCoordinates(ctx, city)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			weather, err := getWeather(ctx, client, coordinates)
			if err == nil {
				fmt.Printf("the Temperature in %s is: %.1f°C\n", city, weather.Temperature)
			}
		case <-ctx.Done():
			return
		}
	}
}

// A Go function that receives a city name and gets it's coordinates (lat and long)
func GetCoordinates(ctx context.Context, city string) (Coordinates, error) {
	baseURL := "https://nominatim.openstreetmap.org/search"
	// Construct the query parameters
	queryParams := url.Values{}
	queryParams.Set("q", city)
	queryParams.Set("format", "json")
	endpointURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())

	// Send a GET request to the API endpoint
	response, err := http.Get(endpointURL)
	if err != nil {
		return Coordinates{}, err
	}
	defer response.Body.Close()

	// Read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Coordinates{}, err
	}

	// Unmarshal the response JSON into a slice of coordinates
	var coordinates []Coordinates
	err = json.Unmarshal(body, &coordinates)
	if err != nil {
		return Coordinates{}, err
	}

	if len(coordinates) == 0 {
		return Coordinates{}, fmt.Errorf("no coordinates found for %s", city)
	}

	// Return the first set of coordinates
	return coordinates[0], nil
}

// Get the current weather for amsterdam
func getWeather(ctx context.Context, c omgo.Client, coord Coordinates) (*omgo.CurrentWeather, error) {

	// Get the humidity and cloud cover forecast for berlin,
	// including the last 2 days and non-metric units
	loc, _ := omgo.NewLocation(coord.Latitude, coord.Longitude)

	opts := omgo.Options{
		TemperatureUnit: "celsius",
		Timezone:        "Asia/Jerusalem",
		PastDays:        2,
		HourlyMetrics:   []string{"cloudcover, relativehumidity_2m"},
		DailyMetrics:    []string{"temperature_2m_max"},
	}

	weather, err := c.CurrentWeather(ctx, loc, &opts)
	if err != nil {
		return nil, err
	}

	return &weather, nil
}

func getRandomCity(countries []Country) string {
	randomIndex := rand.Intn(len(countries))
	capitals := countries[randomIndex].Capital
	if len(capitals) == 0 {
		return "Unknown"
	}
	return capitals[0]
}
