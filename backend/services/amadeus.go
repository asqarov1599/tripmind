package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ─── Types ────────────────────────────────────────────────────────────────────

type Flight struct {
	Price               float64 `json:"price"`
	Airline             string  `json:"airline"`
	AirlineCode         string  `json:"airline_code,omitempty"`
	FlightNumber        string  `json:"flight_number,omitempty"`
	DepartureTime       string  `json:"departure_time"`
	ArrivalTime         string  `json:"arrival_time"`
	Duration            string  `json:"duration"`
	Stops               int     `json:"stops"`
	ReturnDepartureTime string  `json:"return_departure_time,omitempty"`
	ReturnArrivalTime   string  `json:"return_arrival_time,omitempty"`
	ReturnDuration      string  `json:"return_duration,omitempty"`
	ReturnStops         int     `json:"return_stops,omitempty"`
	BookingLink         string  `json:"booking_link,omitempty"`
	Currency            string  `json:"currency,omitempty"`
}

type Hotel struct {
	Name        string  `json:"name"`
	HotelID     string  `json:"hotel_id,omitempty"`
	Price       float64 `json:"price"`
	Rating      float64 `json:"rating"`
	Location    string  `json:"location"`
	BookingLink string  `json:"booking_link,omitempty"`
	Currency    string  `json:"currency,omitempty"`
}

// ─── Amadeus Client ───────────────────────────────────────────────────────────

type AmadeusClient struct {
	clientID     string
	clientSecret string
	baseURL      string
	accessToken  string
	tokenExpiry  time.Time
	mu           sync.Mutex
	httpClient   *http.Client
}

var amadeusClient *AmadeusClient

func InitAmadeus() {
	env := os.Getenv("AMADEUS_ENV")
	baseURL := "https://api.amadeus.com" // production
	if env == "" || env == "test" {
		baseURL = "https://test.api.amadeus.com" // free test environment
	}

	amadeusClient = &AmadeusClient{
		clientID:     os.Getenv("AMADEUS_CLIENT_ID"),
		clientSecret: os.Getenv("AMADEUS_CLIENT_SECRET"),
		baseURL:      baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if amadeusClient.clientID == "" || amadeusClient.clientSecret == "" {
		log.Println("⚠️  AMADEUS_CLIENT_ID or AMADEUS_CLIENT_SECRET not set — flight/hotel search will use fallback data")
		return
	}

	// Pre-warm the token
	if err := amadeusClient.refreshToken(); err != nil {
		log.Printf("⚠️  Amadeus token pre-warm failed: %v", err)
	} else {
		log.Println("✅ Amadeus API authenticated")
	}
}

func GetAmadeusClient() *AmadeusClient {
	return amadeusClient
}

// ─── OAuth2 Token ─────────────────────────────────────────────────────────────

func (c *AmadeusClient) refreshToken() error {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)

	req, err := http.NewRequest("POST",
		c.baseURL+"/v1/security/oauth2/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse token response: %v", err)
	}

	c.mu.Lock()
	c.accessToken = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-30) * time.Second)
	c.mu.Unlock()

	return nil
}

func (c *AmadeusClient) getToken() (string, error) {
	c.mu.Lock()
	expired := time.Now().After(c.tokenExpiry)
	token := c.accessToken
	c.mu.Unlock()

	if expired || token == "" {
		if err := c.refreshToken(); err != nil {
			return "", err
		}
		c.mu.Lock()
		token = c.accessToken
		c.mu.Unlock()
	}
	return token, nil
}

func (c *AmadeusClient) doRequest(method, path string, body []byte) ([]byte, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, fmt.Errorf("auth failed: %w", err)
	}

	var req *http.Request
	if body != nil {
		req, err = http.NewRequest(method, c.baseURL+path, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, c.baseURL+path, nil)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("amadeus error (%d): %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// ─── Flight Search ────────────────────────────────────────────────────────────

// SearchFlights searches real-time flights via Amadeus Flight Offers Search API
func (c *AmadeusClient) SearchFlights(origin, destination, departureDate, returnDate string, adults int) ([]Flight, error) {
	if c.clientID == "" {
		return nil, fmt.Errorf("amadeus not configured")
	}

	path := fmt.Sprintf(
		"/v2/shopping/flight-offers?originLocationCode=%s&destinationLocationCode=%s"+
			"&departureDate=%s&returnDate=%s&adults=%d&max=6&currencyCode=USD",
		url.QueryEscape(origin),
		url.QueryEscape(destination),
		url.QueryEscape(departureDate),
		url.QueryEscape(returnDate),
		adults,
	)

	body, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("flight search failed: %w", err)
	}

	return parseFlightOffers(body)
}

// Amadeus flight offers response structures
type amadeusFlightOffersResponse struct {
	Data []amadeusFlightOffer `json:"data"`
}

type amadeusFlightOffer struct {
	Price struct {
		GrandTotal string `json:"grandTotal"`
		Currency   string `json:"currency"`
	} `json:"price"`
	Itineraries []struct {
		Duration string `json:"duration"`
		Segments []struct {
			Departure struct {
				IataCode string `json:"iataCode"`
				At       string `json:"at"`
			} `json:"departure"`
			Arrival struct {
				IataCode string `json:"iataCode"`
				At       string `json:"at"`
			} `json:"arrival"`
			CarrierCode string `json:"carrierCode"`
			Number      string `json:"number"`
		} `json:"segments"`
	} `json:"itineraries"`
	ValidatingAirlineCodes []string `json:"validatingAirlineCodes"`
}

func parseFlightOffers(data []byte) ([]Flight, error) {
	var resp amadeusFlightOffersResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse flight offers: %w", err)
	}

	flights := make([]Flight, 0, len(resp.Data))

	for _, offer := range resp.Data {
		if len(offer.Itineraries) < 1 {
			continue
		}

		price := parsePrice(offer.Price.GrandTotal)
		if price <= 0 {
			continue
		}

		outbound := offer.Itineraries[0]
		var returnIt *struct {
			Duration string `json:"duration"`
			Segments []struct {
				Departure struct {
					IataCode string `json:"iataCode"`
					At       string `json:"at"`
				} `json:"departure"`
				Arrival struct {
					IataCode string `json:"iataCode"`
					At       string `json:"at"`
				} `json:"arrival"`
				CarrierCode string `json:"carrierCode"`
				Number      string `json:"number"`
			} `json:"segments"`
		}
		if len(offer.Itineraries) >= 2 {
			it := offer.Itineraries[1]
			returnIt = &it
		}

		airlineCode := ""
		if len(outbound.Segments) > 0 {
			airlineCode = outbound.Segments[0].CarrierCode
		} else if len(offer.ValidatingAirlineCodes) > 0 {
			airlineCode = offer.ValidatingAirlineCodes[0]
		}

		f := Flight{
			Price:       price,
			Airline:     airlineName(airlineCode),
			AirlineCode: airlineCode,
			Currency:    offer.Price.Currency,
			Stops:       max(0, len(outbound.Segments)-1),
			Duration:    parseDuration(outbound.Duration),
		}

		if len(outbound.Segments) > 0 {
			f.DepartureTime = outbound.Segments[0].Departure.At
			f.ArrivalTime = outbound.Segments[len(outbound.Segments)-1].Arrival.At
			f.FlightNumber = airlineCode + outbound.Segments[0].Number
		}

		if returnIt != nil {
			f.ReturnStops = max(0, len(returnIt.Segments)-1)
			f.ReturnDuration = parseDuration(returnIt.Duration)
			if len(returnIt.Segments) > 0 {
				f.ReturnDepartureTime = returnIt.Segments[0].Departure.At
				f.ReturnArrivalTime = returnIt.Segments[len(returnIt.Segments)-1].Arrival.At
			}
		}

		flights = append(flights, f)
	}

	return flights, nil
}

// ─── Hotel Search ─────────────────────────────────────────────────────────────

// SearchHotels searches hotels via Amadeus Hotel List + Hotel Offers APIs
func (c *AmadeusClient) SearchHotels(cityCode, checkIn, checkOut string, adults int) ([]Hotel, error) {
	if c.clientID == "" {
		return nil, fmt.Errorf("amadeus not configured")
	}

	// Step 1: Get hotel IDs for the city
	hotelIDs, err := c.getHotelIDsByCity(cityCode)
	if err != nil {
		return nil, fmt.Errorf("hotel list failed: %w", err)
	}

	if len(hotelIDs) == 0 {
		return nil, fmt.Errorf("no hotels found for city %s", cityCode)
	}

	// Limit to first 20 IDs to avoid hitting rate limits
	if len(hotelIDs) > 20 {
		hotelIDs = hotelIDs[:20]
	}

	// Step 2: Get available offers for those hotels
	return c.getHotelOffers(hotelIDs, checkIn, checkOut, adults)
}

type amadeusHotelListResponse struct {
	Data []struct {
		HotelID  string `json:"hotelId"`
		Name     string `json:"name"`
		IATACode string `json:"iataCode"`
		GeoCode  struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"geoCode"`
		Address struct {
			CountryCode string `json:"countryCode"`
		} `json:"address"`
	} `json:"data"`
}

func (c *AmadeusClient) getHotelIDsByCity(cityCode string) ([]string, error) {
	// Airport IATA codes map to city codes for hotel search
	// Amadeus uses city codes, not airport codes for hotel search
	hotelCityCode := airportToCity(cityCode)

	path := fmt.Sprintf("/v1/reference-data/locations/hotels/by-city?cityCode=%s&radius=5&radiusUnit=KM&hotelSource=ALL",
		url.QueryEscape(hotelCityCode))

	body, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp amadeusHotelListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse hotel list: %w", err)
	}

	ids := make([]string, 0, len(resp.Data))
	for _, h := range resp.Data {
		ids = append(ids, h.HotelID)
	}
	return ids, nil
}

type amadeusHotelOffersResponse struct {
	Data []struct {
		Hotel struct {
			HotelID string `json:"hotelId"`
			Name    string `json:"name"`
			CityCode string `json:"cityCode"`
			Address struct {
				Lines       []string `json:"lines"`
				CityName    string   `json:"cityName"`
				CountryCode string   `json:"countryCode"`
			} `json:"address"`
			Rating string `json:"rating"`
		} `json:"hotel"`
		Available bool `json:"available"`
		Offers    []struct {
			Price struct {
				Total    string `json:"total"`
				Currency string `json:"currency"`
			} `json:"price"`
		} `json:"offers"`
	} `json:"data"`
}

func (c *AmadeusClient) getHotelOffers(hotelIDs []string, checkIn, checkOut string, adults int) ([]Hotel, error) {
	path := fmt.Sprintf("/v3/shopping/hotel-offers?hotelIds=%s&checkInDate=%s&checkOutDate=%s&adults=%d&roomQuantity=1&currency=USD&bestRateOnly=true",
		url.QueryEscape(strings.Join(hotelIDs, ",")),
		url.QueryEscape(checkIn),
		url.QueryEscape(checkOut),
		adults,
	)

	body, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("hotel offers failed: %w", err)
	}

	var resp amadeusHotelOffersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse hotel offers: %w", err)
	}

	hotels := make([]Hotel, 0, len(resp.Data))
	for _, item := range resp.Data {
		if !item.Available || len(item.Offers) == 0 {
			continue
		}

		price := parsePrice(item.Offers[0].Price.Total)
		if price <= 0 {
			continue
		}

		rating := parseRating(item.Hotel.Rating)
		location := item.Hotel.Address.CityName
		if location == "" {
			location = item.Hotel.CityCode
		}

		hotels = append(hotels, Hotel{
			Name:     item.Hotel.Name,
			HotelID:  item.Hotel.HotelID,
			Price:    price,
			Rating:   rating,
			Location: location,
			Currency: item.Offers[0].Price.Currency,
		})
	}

	return hotels, nil
}

// ─── Fallback (when Amadeus is not configured or fails) ──────────────────────

// GenerateFlightsFallback produces plausible flight data without an API key.
// This is clearly labeled as estimated data in the AI summary.
func GenerateFlightsFallback(origin, destination, departureDate, returnDate string) []Flight {
	type routeInfo struct {
		basePrice float64
		duration  int // minutes
	}

	routes := map[string]routeInfo{
		"TAS-IST": {280, 300}, "IST-TAS": {280, 300},
		"TAS-DXB": {320, 210}, "DXB-TAS": {320, 210},
		"TAS-FRA": {450, 420}, "FRA-TAS": {450, 420},
		"TAS-LHR": {500, 480}, "LHR-TAS": {500, 480},
		"TAS-CDG": {480, 450}, "CDG-TAS": {480, 450},
		"BER-PAR": {120, 105}, "PAR-BER": {120, 105},
		"BER-LHR": {100, 100}, "LHR-BER": {100, 100},
		"IST-DXB": {250, 240}, "DXB-IST": {250, 240},
		"LHR-JFK": {450, 480}, "JFK-LHR": {450, 480},
		"LHR-CDG": {80, 75},  "CDG-LHR": {80, 75},
		"FRA-IST": {150, 165}, "IST-FRA": {150, 165},
	}

	key := origin + "-" + destination
	info, ok := routes[key]
	if !ok {
		info = routeInfo{350, 240}
	}

	// Five airline options across price tiers
	type airlineOption struct {
		name     string
		priceMod float64
		stops    int
	}
	options := []airlineOption{
		{"Turkish Airlines", 1.00, 0},
		{"Lufthansa", 1.15, 0},
		{"Emirates", 1.30, 0},
		{"Wizz Air", 0.65, 1},
		{"FlyDubai", 0.80, 1},
	}

	depDate, _ := time.Parse("2006-01-02", departureDate)
	retDate, _ := time.Parse("2006-01-02", returnDate)

	flights := make([]Flight, 0, len(options))
	for i, opt := range options {
		price := info.basePrice * opt.priceMod
		price = float64(int(price/5)*5)

		dur := info.duration
		if opt.stops > 0 {
			dur += 90
		}

		depHour := 6 + i*3
		retHour := 8 + i*2

		depTime := time.Date(depDate.Year(), depDate.Month(), depDate.Day(), depHour, 0, 0, 0, time.UTC)
		arrTime := depTime.Add(time.Duration(dur) * time.Minute)
		retDepTime := time.Date(retDate.Year(), retDate.Month(), retDate.Day(), retHour, 0, 0, 0, time.UTC)
		retArrTime := retDepTime.Add(time.Duration(dur) * time.Minute)

		flights = append(flights, Flight{
			Price:               price,
			Airline:             opt.name,
			DepartureTime:       depTime.Format(time.RFC3339),
			ArrivalTime:         arrTime.Format(time.RFC3339),
			Duration:            formatDurationMin(dur),
			Stops:               opt.stops,
			ReturnDepartureTime: retDepTime.Format(time.RFC3339),
			ReturnArrivalTime:   retArrTime.Format(time.RFC3339),
			ReturnDuration:      formatDurationMin(dur),
			ReturnStops:         opt.stops,
		})
	}
	return flights
}

// GenerateHotelsFallback produces plausible hotel data without an API key.
func GenerateHotelsFallback(destination string) []Hotel {
	cityHotels := map[string][]Hotel{
		"IST": {
			{"Grand Hyatt Istanbul", "", 180, 4.7, "Beyoglu, Istanbul", "", "USD"},
			{"Hilton Istanbul Bosphorus", "", 165, 4.5, "Besiktas, Istanbul", "", "USD"},
			{"Sultan Ahmet Palace Hotel", "", 95, 4.3, "Sultanahmet, Istanbul", "", "USD"},
			{"Ibis Istanbul Taksim", "", 75, 4.0, "Taksim, Istanbul", "", "USD"},
			{"The Marmara Taksim", "", 140, 4.4, "Taksim Square, Istanbul", "", "USD"},
		},
		"CDG": { // Paris
			{"Hotel Le Marais", "", 220, 4.6, "Le Marais, Paris", "", "USD"},
			{"Pullman Paris Tour Eiffel", "", 280, 4.5, "7th Arr., Paris", "", "USD"},
			{"Ibis Paris Montmartre", "", 95, 4.0, "Montmartre, Paris", "", "USD"},
			{"Hotel des Arts Montmartre", "", 130, 4.3, "18th Arr., Paris", "", "USD"},
			{"Generator Paris", "", 55, 3.8, "10th Arr., Paris", "", "USD"},
		},
		"PAR": { // Also try PAR for Paris
			{"Hotel Le Marais", "", 220, 4.6, "Le Marais, Paris", "", "USD"},
			{"Pullman Paris Tour Eiffel", "", 280, 4.5, "7th Arr., Paris", "", "USD"},
			{"Ibis Paris Montmartre", "", 95, 4.0, "Montmartre, Paris", "", "USD"},
			{"Hotel des Arts Montmartre", "", 130, 4.3, "18th Arr., Paris", "", "USD"},
		},
		"LHR": { // London
			{"Hilton London Tower Bridge", "", 180, 4.4, "Tower Bridge, London", "", "USD"},
			{"Premier Inn London City", "", 95, 4.1, "City of London", "", "USD"},
			{"The Hoxton Shoreditch", "", 165, 4.5, "Shoreditch, London", "", "USD"},
			{"Generator London", "", 50, 3.8, "Russell Square, London", "", "USD"},
			{"citizenM London Bankside", "", 145, 4.4, "Bankside, London", "", "USD"},
		},
		"DXB": { // Dubai
			{"JW Marriott Marquis", "", 220, 4.6, "Business Bay, Dubai", "", "USD"},
			{"Rove Downtown", "", 95, 4.3, "Downtown Dubai", "", "USD"},
			{"Premier Inn Dubai", "", 65, 4.0, "Ibn Battuta, Dubai", "", "USD"},
			{"Atlantis The Palm", "", 380, 4.7, "Palm Jumeirah, Dubai", "", "USD"},
			{"Hilton Dubai Al Habtoor City", "", 160, 4.4, "Dubai Marina", "", "USD"},
		},
		"FRA": { // Frankfurt
			{"Marriott Frankfurt City Center", "", 155, 4.4, "Sachsenhausen, Frankfurt", "", "USD"},
			{"Motel One Frankfurt-Römer", "", 89, 4.3, "Römer, Frankfurt", "", "USD"},
			{"Hilton Frankfurt City Centre", "", 175, 4.5, "City Centre, Frankfurt", "", "USD"},
			{"Generator Frankfurt", "", 45, 3.9, "Sachsenhausen, Frankfurt", "", "USD"},
			{"Steigenberger Frankfurter Hof", "", 280, 4.6, "Kaiserplatz, Frankfurt", "", "USD"},
		},
		"BER": { // Berlin
			{"Hotel Adlon Kempinski", "", 320, 4.8, "Mitte, Berlin", "", "USD"},
			{"Radisson Blu Berlin", "", 150, 4.4, "Alexanderplatz, Berlin", "", "USD"},
			{"Motel One Berlin Hackescher Markt", "", 85, 4.2, "Mitte, Berlin", "", "USD"},
			{"Generator Berlin Mitte", "", 45, 3.9, "Mitte, Berlin", "", "USD"},
			{"Michelberger Hotel", "", 130, 4.5, "Friedrichshain, Berlin", "", "USD"},
		},
	}

	if hotels, ok := cityHotels[destination]; ok {
		return hotels
	}

	// Generic fallback
	return []Hotel{
		{"Grand City Hotel", "", 150, 4.5, "City Center, " + destination, "", "USD"},
		{"Business Inn", "", 95, 4.2, "Business District, " + destination, "", "USD"},
		{"Boutique Residence", "", 120, 4.4, "Arts District, " + destination, "", "USD"},
		{"Economy Suites", "", 65, 3.9, "Near Airport, " + destination, "", "USD"},
		{"Luxury Collection", "", 240, 4.7, "Historic Center, " + destination, "", "USD"},
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// parseDuration converts ISO 8601 duration (PT5H30M) to human readable (5h 30m)
func parseDuration(iso string) string {
	if iso == "" {
		return ""
	}
	iso = strings.TrimPrefix(iso, "PT")
	result := ""
	hIdx := strings.Index(iso, "H")
	mIdx := strings.Index(iso, "M")
	if hIdx >= 0 {
		result += iso[:hIdx] + "h"
		iso = iso[hIdx+1:]
		mIdx = strings.Index(iso, "M")
	}
	if mIdx >= 0 && mIdx < len(iso) {
		if result != "" {
			result += " "
		}
		result += iso[:mIdx] + "m"
	}
	return result
}

func formatDurationMin(minutes int) string {
	h := minutes / 60
	m := minutes % 60
	if m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dh", h)
}

func parsePrice(s string) float64 {
	var price float64
	fmt.Sscanf(s, "%f", &price)
	return price
}

func parseRating(s string) float64 {
	if s == "" {
		return 4.0
	}
	var r float64
	fmt.Sscanf(s, "%f", &r)
	if r <= 0 {
		return 4.0
	}
	// Amadeus returns star ratings 1-5
	if r > 5 {
		r = 5
	}
	return r
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// airportToCity maps airport IATA codes to city codes for hotel search
func airportToCity(airport string) string {
	mapping := map[string]string{
		"LHR": "LON", "LGW": "LON", "STN": "LON", "LTN": "LON",
		"CDG": "PAR", "ORY": "PAR",
		"JFK": "NYC", "LGA": "NYC", "EWR": "NYC",
		"LAX": "LAX",
		"DXB": "DXB",
		"IST": "IST",
		"FRA": "FRA",
		"AMS": "AMS",
		"BER": "BER", "SXF": "BER",
		"MAD": "MAD",
		"BCN": "BCN",
		"FCO": "ROM", "CIA": "ROM",
		"TAS": "TAS",
		"NRT": "TYO", "HND": "TYO",
		"SIN": "SIN",
		"BKK": "BKK",
	}
	if city, ok := mapping[airport]; ok {
		return city
	}
	return airport // fallback: use as-is
}

// airlineName returns full airline name from IATA code
func airlineName(code string) string {
	names := map[string]string{
		"TK": "Turkish Airlines",
		"LH": "Lufthansa",
		"AF": "Air France",
		"BA": "British Airways",
		"EK": "Emirates",
		"QR": "Qatar Airways",
		"PC": "Pegasus Airlines",
		"FR": "Ryanair",
		"U2": "EasyJet",
		"W6": "Wizz Air",
		"FZ": "FlyDubai",
		"HY": "Uzbekistan Airways",
		"UA": "United Airlines",
		"AA": "American Airlines",
		"DL": "Delta Air Lines",
		"KL": "KLM",
		"IB": "Iberia",
		"AZ": "ITA Airways",
		"OS": "Austrian Airlines",
		"LX": "Swiss International Air Lines",
		"SQ": "Singapore Airlines",
		"CX": "Cathay Pacific",
		"NH": "ANA",
		"JL": "Japan Airlines",
		"EY": "Etihad Airways",
		"SV": "Saudi Arabian Airlines",
		"MS": "EgyptAir",
		"RJ": "Royal Jordanian",
		"ET": "Ethiopian Airlines",
		"KQ": "Kenya Airways",
		"SA": "South African Airways",
	}
	if name, ok := names[code]; ok {
		return name
	}
	if code != "" {
		return code + " Airlines"
	}
	return "Unknown Airline"
}

// FallbackRecommendation provides basic recommendation when AI fails
func FallbackRecommendation(budget float64, flights []Flight, hotels []Hotel, numNights int) string {
	if len(flights) == 0 || len(hotels) == 0 {
		return "Unable to provide recommendations at this time."
	}

	cheapestFlight := flights[0]
	for _, f := range flights {
		if f.Price < cheapestFlight.Price {
			cheapestFlight = f
		}
	}

	bestValueHotel := hotels[0]
	for _, h := range hotels {
		if h.Price < bestValueHotel.Price {
			bestValueHotel = h
		}
	}

	total := cheapestFlight.Price + bestValueHotel.Price*float64(numNights)
	withinBudget := ""
	if total <= budget {
		withinBudget = fmt.Sprintf(" This combination fits your $%.0f budget.", budget)
	} else {
		withinBudget = fmt.Sprintf(" Note: This exceeds your $%.0f budget by $%.0f.", budget, total-budget)
	}

	return fmt.Sprintf(
		"Best value picks: %s at $%.0f (%.0f stops) and %s at $%.0f/night (★ %.1f). "+
			"Estimated total: $%.0f for flight + %d nights.%s",
		cheapestFlight.Airline, cheapestFlight.Price, float64(cheapestFlight.Stops),
		bestValueHotel.Name, bestValueHotel.Price, bestValueHotel.Rating,
		total, numNights, withinBudget,
	)
}
