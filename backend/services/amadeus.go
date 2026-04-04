package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
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
	baseURL := "https://api.amadeus.com"
	if env == "" || env == "test" {
		baseURL = "https://test.api.amadeus.com"
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
		log.Println("⚠️  AMADEUS_CLIENT_ID or AMADEUS_CLIENT_SECRET not set — using rich mock data")
		return
	}

	if err := amadeusClient.refreshToken(); err != nil {
		log.Printf("⚠️  Amadeus token failed: %v — using rich mock data instead", err)
	} else {
		log.Println("✅ Amadeus API authenticated")
	}
}

func GetAmadeusClient() *AmadeusClient {
	return amadeusClient
}

func (c *AmadeusClient) refreshToken() error {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)

	req, err := http.NewRequest("POST", c.baseURL+"/v1/security/oauth2/token", strings.NewReader(form.Encode()))
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

func (c *AmadeusClient) SearchFlights(origin, destination, departureDate, returnDate string, adults int) ([]Flight, error) {
	if c.clientID == "" {
		return nil, fmt.Errorf("amadeus not configured")
	}

	path := fmt.Sprintf(
		"/v2/shopping/flight-offers?originLocationCode=%s&destinationLocationCode=%s&departureDate=%s&returnDate=%s&adults=%d&max=6&currencyCode=USD",
		url.QueryEscape(origin), url.QueryEscape(destination),
		url.QueryEscape(departureDate), url.QueryEscape(returnDate), adults,
	)

	body, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("flight search failed: %w", err)
	}

	return parseFlightOffers(body)
}

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
			Departure   struct{ IataCode, At string } `json:"departure"`
			Arrival     struct{ IataCode, At string } `json:"arrival"`
			CarrierCode string                        `json:"carrierCode"`
			Number      string                        `json:"number"`
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

		if len(offer.Itineraries) >= 2 {
			ret := offer.Itineraries[1]
			f.ReturnStops = max(0, len(ret.Segments)-1)
			f.ReturnDuration = parseDuration(ret.Duration)
			if len(ret.Segments) > 0 {
				f.ReturnDepartureTime = ret.Segments[0].Departure.At
				f.ReturnArrivalTime = ret.Segments[len(ret.Segments)-1].Arrival.At
			}
		}

		flights = append(flights, f)
	}
	return flights, nil
}

// ─── Hotel Search ─────────────────────────────────────────────────────────────

func (c *AmadeusClient) SearchHotels(cityCode, checkIn, checkOut string, adults int) ([]Hotel, error) {
	if c.clientID == "" {
		return nil, fmt.Errorf("amadeus not configured")
	}

	hotelIDs, err := c.getHotelIDsByCity(cityCode)
	if err != nil {
		return nil, fmt.Errorf("hotel list failed: %w", err)
	}
	if len(hotelIDs) == 0 {
		return nil, fmt.Errorf("no hotels found for city %s", cityCode)
	}
	if len(hotelIDs) > 20 {
		hotelIDs = hotelIDs[:20]
	}
	return c.getHotelOffers(hotelIDs, checkIn, checkOut, adults)
}

type amadeusHotelListResponse struct {
	Data []struct {
		HotelID string `json:"hotelId"`
	} `json:"data"`
}

func (c *AmadeusClient) getHotelIDsByCity(cityCode string) ([]string, error) {
	hotelCityCode := airportToCity(cityCode)
	path := fmt.Sprintf("/v1/reference-data/locations/hotels/by-city?cityCode=%s&radius=5&radiusUnit=KM&hotelSource=ALL", url.QueryEscape(hotelCityCode))

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
			HotelID  string `json:"hotelId"`
			Name     string `json:"name"`
			CityCode string `json:"cityCode"`
			Address  struct {
				CityName    string `json:"cityName"`
				CountryCode string `json:"countryCode"`
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
		url.QueryEscape(checkIn), url.QueryEscape(checkOut), adults,
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
		location := item.Hotel.Address.CityName
		if location == "" {
			location = item.Hotel.CityCode
		}
		hotels = append(hotels, Hotel{
			Name:     item.Hotel.Name,
			HotelID:  item.Hotel.HotelID,
			Price:    price,
			Rating:   parseRating(item.Hotel.Rating),
			Location: location,
			Currency: item.Offers[0].Price.Currency,
		})
	}
	return hotels, nil
}

// ─── Rich Fallback Data ───────────────────────────────────────────────────────

type routeData struct {
	basePrice int
	durationM int
	airlines  []fallbackAirline
}

type fallbackAirline struct {
	name        string
	code        string
	flightNum   string
	priceFactor float64
	stops       int
	depHour     int
	retHour     int
}

var knownRoutes = map[string]routeData{
	"TAS-IST": {310, 295, []fallbackAirline{
		{"Turkish Airlines", "TK", "TK315", 1.00, 0, 6, 14},
		{"Uzbekistan Airways", "HY", "HY122", 0.90, 0, 9, 10},
		{"Pegasus Airlines", "PC", "PC960", 0.75, 1, 5, 18},
		{"FlyDubai", "FZ", "FZ901", 0.82, 1, 23, 7},
		{"Air Arabia", "G9", "G9412", 0.70, 1, 2, 6},
	}},
	"IST-TAS": {310, 295, []fallbackAirline{
		{"Turkish Airlines", "TK", "TK316", 1.00, 0, 14, 6},
		{"Uzbekistan Airways", "HY", "HY121", 0.90, 0, 10, 9},
		{"Pegasus Airlines", "PC", "PC961", 0.75, 1, 18, 5},
		{"FlyDubai", "FZ", "FZ902", 0.82, 1, 7, 23},
		{"Air Arabia", "G9", "G9413", 0.70, 1, 6, 2},
	}},
	"TAS-DXB": {340, 205, []fallbackAirline{
		{"Emirates", "EK", "EK872", 1.20, 0, 8, 15},
		{"Uzbekistan Airways", "HY", "HY534", 0.90, 0, 11, 12},
		{"FlyDubai", "FZ", "FZ531", 0.78, 0, 6, 19},
		{"Air Arabia", "G9", "G9220", 0.72, 1, 4, 8},
		{"Wizz Air", "W6", "W68801", 0.68, 1, 2, 22},
	}},
	"DXB-TAS": {340, 205, []fallbackAirline{
		{"Emirates", "EK", "EK871", 1.20, 0, 15, 8},
		{"Uzbekistan Airways", "HY", "HY535", 0.90, 0, 12, 11},
		{"FlyDubai", "FZ", "FZ532", 0.78, 0, 19, 6},
		{"Air Arabia", "G9", "G9221", 0.72, 1, 8, 4},
		{"Wizz Air", "W6", "W68802", 0.68, 1, 22, 2},
	}},
	"TAS-FRA": {510, 415, []fallbackAirline{
		{"Lufthansa", "LH", "LH1536", 1.10, 1, 7, 16},
		{"Turkish Airlines", "TK", "TK0027", 1.00, 1, 6, 14},
		{"Uzbekistan Airways", "HY", "HY014", 0.88, 0, 10, 9},
		{"Qatar Airways", "QR", "QR543", 1.15, 1, 4, 7},
		{"Emirates", "EK", "EK033", 1.25, 1, 2, 5},
	}},
	"FRA-TAS": {510, 415, []fallbackAirline{
		{"Lufthansa", "LH", "LH1537", 1.10, 1, 16, 7},
		{"Turkish Airlines", "TK", "TK0028", 1.00, 1, 14, 6},
		{"Uzbekistan Airways", "HY", "HY015", 0.88, 0, 9, 10},
		{"Qatar Airways", "QR", "QR544", 1.15, 1, 7, 4},
		{"Emirates", "EK", "EK034", 1.25, 1, 5, 2},
	}},
	"TAS-LHR": {555, 475, []fallbackAirline{
		{"British Airways", "BA", "BA0879", 1.15, 1, 7, 16},
		{"Turkish Airlines", "TK", "TK0019", 1.00, 1, 5, 14},
		{"Uzbekistan Airways", "HY", "HY002", 0.90, 1, 10, 9},
		{"Qatar Airways", "QR", "QR003", 1.20, 1, 3, 7},
		{"Emirates", "EK", "EK021", 1.30, 1, 1, 5},
	}},
	"LHR-TAS": {555, 475, []fallbackAirline{
		{"British Airways", "BA", "BA0880", 1.15, 1, 16, 7},
		{"Turkish Airlines", "TK", "TK0020", 1.00, 1, 14, 5},
		{"Uzbekistan Airways", "HY", "HY003", 0.90, 1, 9, 10},
		{"Qatar Airways", "QR", "QR004", 1.20, 1, 7, 3},
		{"Emirates", "EK", "EK022", 1.30, 1, 5, 1},
	}},
	"TAS-CDG": {490, 450, []fallbackAirline{
		{"Air France", "AF", "AF0774", 1.10, 1, 7, 15},
		{"Turkish Airlines", "TK", "TK0025", 1.00, 1, 5, 14},
		{"Uzbekistan Airways", "HY", "HY010", 0.88, 1, 10, 9},
		{"Emirates", "EK", "EK073", 1.25, 1, 2, 4},
		{"Qatar Airways", "QR", "QR039", 1.15, 1, 4, 6},
	}},
	"CDG-TAS": {490, 450, []fallbackAirline{
		{"Air France", "AF", "AF0775", 1.10, 1, 15, 7},
		{"Turkish Airlines", "TK", "TK0026", 1.00, 1, 14, 5},
		{"Uzbekistan Airways", "HY", "HY011", 0.88, 1, 9, 10},
		{"Emirates", "EK", "EK074", 1.25, 1, 4, 2},
		{"Qatar Airways", "QR", "QR040", 1.15, 1, 6, 4},
	}},
	"LHR-CDG": {88, 75, []fallbackAirline{
		{"British Airways", "BA", "BA0304", 1.10, 0, 7, 17},
		{"Air France", "AF", "AF1681", 1.05, 0, 9, 18},
		{"EasyJet", "U2", "U22001", 0.72, 0, 6, 20},
		{"Vueling", "VY", "VY8808", 0.68, 0, 11, 14},
		{"Iberia", "IB", "IB3172", 0.75, 0, 12, 8},
	}},
	"CDG-LHR": {88, 75, []fallbackAirline{
		{"Air France", "AF", "AF1682", 1.05, 0, 17, 9},
		{"British Airways", "BA", "BA0303", 1.10, 0, 18, 7},
		{"EasyJet", "U2", "U22002", 0.72, 0, 20, 6},
		{"Vueling", "VY", "VY8809", 0.68, 0, 14, 11},
		{"Iberia", "IB", "IB3171", 0.75, 0, 8, 12},
	}},
	"BER-LHR": {108, 100, []fallbackAirline{
		{"British Airways", "BA", "BA0984", 1.10, 0, 7, 18},
		{"Ryanair", "FR", "FR0023", 0.65, 0, 6, 20},
		{"EasyJet", "U2", "U26701", 0.72, 0, 9, 17},
		{"Lufthansa", "LH", "LH0932", 1.15, 0, 8, 15},
		{"Wizz Air", "W6", "W63301", 0.60, 0, 5, 22},
	}},
	"LHR-BER": {108, 100, []fallbackAirline{
		{"British Airways", "BA", "BA0983", 1.10, 0, 18, 7},
		{"Ryanair", "FR", "FR0024", 0.65, 0, 20, 6},
		{"EasyJet", "U2", "U26702", 0.72, 0, 17, 9},
		{"Lufthansa", "LH", "LH0931", 1.15, 0, 15, 8},
		{"Wizz Air", "W6", "W63302", 0.60, 0, 22, 5},
	}},
	"BER-CDG": {95, 105, []fallbackAirline{
		{"Air France", "AF", "AF1235", 1.08, 0, 7, 17},
		{"Lufthansa", "LH", "LH1034", 1.12, 0, 9, 16},
		{"EasyJet", "U2", "U29902", 0.70, 0, 6, 19},
		{"Transavia", "HV", "HV5401", 0.75, 0, 8, 14},
		{"Ryanair", "FR", "FR1256", 0.62, 0, 5, 21},
	}},
	"CDG-BER": {95, 105, []fallbackAirline{
		{"Air France", "AF", "AF1236", 1.08, 0, 17, 7},
		{"Lufthansa", "LH", "LH1035", 1.12, 0, 16, 9},
		{"EasyJet", "U2", "U29903", 0.70, 0, 19, 6},
		{"Transavia", "HV", "HV5402", 0.75, 0, 14, 8},
		{"Ryanair", "FR", "FR1257", 0.62, 0, 21, 5},
	}},
	"LHR-JFK": {490, 430, []fallbackAirline{
		{"British Airways", "BA", "BA0117", 1.12, 0, 11, 18},
		{"American Airlines", "AA", "AA0106", 1.08, 0, 10, 20},
		{"Virgin Atlantic", "VS", "VS0025", 1.05, 0, 12, 17},
		{"United Airlines", "UA", "UA0016", 1.00, 0, 9, 21},
		{"Norse Atlantic", "N0", "N00001", 0.72, 0, 8, 14},
	}},
	"JFK-LHR": {490, 430, []fallbackAirline{
		{"British Airways", "BA", "BA0118", 1.12, 0, 18, 11},
		{"American Airlines", "AA", "AA0107", 1.08, 0, 20, 10},
		{"Virgin Atlantic", "VS", "VS0026", 1.05, 0, 17, 12},
		{"United Airlines", "UA", "UA0017", 1.00, 0, 21, 9},
		{"Norse Atlantic", "N0", "N00002", 0.72, 0, 14, 8},
	}},
	"IST-DXB": {270, 240, []fallbackAirline{
		{"Emirates", "EK", "EK119", 1.18, 0, 8, 15},
		{"Turkish Airlines", "TK", "TK0760", 1.00, 0, 6, 14},
		{"FlyDubai", "FZ", "FZ701", 0.78, 0, 10, 19},
		{"Pegasus Airlines", "PC", "PC512", 0.70, 0, 5, 21},
		{"Air Arabia", "G9", "G9201", 0.68, 1, 2, 7},
	}},
	"DXB-IST": {270, 240, []fallbackAirline{
		{"Emirates", "EK", "EK120", 1.18, 0, 15, 8},
		{"Turkish Airlines", "TK", "TK0759", 1.00, 0, 14, 6},
		{"FlyDubai", "FZ", "FZ702", 0.78, 0, 19, 10},
		{"Pegasus Airlines", "PC", "PC513", 0.70, 0, 21, 5},
		{"Air Arabia", "G9", "G9202", 0.68, 1, 7, 2},
	}},
	"FRA-IST": {162, 165, []fallbackAirline{
		{"Turkish Airlines", "TK", "TK1582", 1.00, 0, 7, 17},
		{"Lufthansa", "LH", "LH1304", 1.15, 0, 9, 16},
		{"Pegasus Airlines", "PC", "PC786", 0.72, 0, 6, 19},
		{"SunExpress", "XQ", "XQ103", 0.68, 0, 5, 21},
		{"Wizz Air", "W6", "W64501", 0.60, 0, 4, 22},
	}},
	"IST-FRA": {162, 165, []fallbackAirline{
		{"Turkish Airlines", "TK", "TK1581", 1.00, 0, 17, 7},
		{"Lufthansa", "LH", "LH1303", 1.15, 0, 16, 9},
		{"Pegasus Airlines", "PC", "PC785", 0.72, 0, 19, 6},
		{"SunExpress", "XQ", "XQ104", 0.68, 0, 21, 5},
		{"Wizz Air", "W6", "W64502", 0.60, 0, 22, 4},
	}},
	"JFK-CDG": {520, 425, []fallbackAirline{
		{"Air France", "AF", "AF0011", 1.10, 0, 18, 10},
		{"American Airlines", "AA", "AA0043", 1.05, 0, 17, 11},
		{"Delta", "DL", "DL0264", 1.08, 0, 19, 12},
		{"Norse Atlantic", "N0", "N00002", 0.70, 0, 20, 9},
		{"United Airlines", "UA", "UA0087", 1.00, 0, 21, 8},
	}},
	"CDG-JFK": {520, 425, []fallbackAirline{
		{"Air France", "AF", "AF0012", 1.10, 0, 10, 18},
		{"American Airlines", "AA", "AA0044", 1.05, 0, 11, 17},
		{"Delta", "DL", "DL0263", 1.08, 0, 12, 19},
		{"Norse Atlantic", "N0", "N00001", 0.70, 0, 9, 20},
		{"United Airlines", "UA", "UA0088", 1.00, 0, 8, 21},
	}},
	"LHR-BKK": {690, 680, []fallbackAirline{
		{"Thai Airways", "TG", "TG0917", 1.10, 1, 21, 12},
		{"British Airways", "BA", "BA0009", 1.15, 1, 22, 11},
		{"Qatar Airways", "QR", "QR0811", 1.20, 1, 20, 13},
		{"Emirates", "EK", "EK0085", 1.25, 1, 19, 14},
		{"Finnair", "AY", "AY0137", 1.00, 1, 23, 10},
	}},
	"LHR-SIN": {720, 740, []fallbackAirline{
		{"Singapore Airlines", "SQ", "SQ0321", 1.20, 0, 21, 14},
		{"British Airways", "BA", "BA0011", 1.15, 1, 22, 12},
		{"Qatar Airways", "QR", "QR0007", 1.18, 1, 20, 11},
		{"Emirates", "EK", "EK0003", 1.22, 1, 19, 15},
		{"Scoot", "TR", "TR0727", 0.72, 1, 23, 9},
	}},
}

// GenerateFlightsFallback produces highly realistic flight data without an API key.
func GenerateFlightsFallback(origin, destination, departureDate, returnDate string) []Flight {
	key := origin + "-" + destination
	route, ok := knownRoutes[key]
	if !ok {
		route = estimateRoute(origin, destination)
	}

	depDate, _ := time.Parse("2006-01-02", departureDate)
	retDate, _ := time.Parse("2006-01-02", returnDate)

	flights := make([]Flight, 0, len(route.airlines))
	for _, opt := range route.airlines {
		dur := route.durationM
		if opt.stops > 0 {
			dur += 85
		}
		price := math.Round(float64(route.basePrice)*opt.priceFactor/5) * 5

		depTime := time.Date(depDate.Year(), depDate.Month(), depDate.Day(), opt.depHour, 25, 0, 0, time.UTC)
		arrTime := depTime.Add(time.Duration(dur) * time.Minute)
		retDepTime := time.Date(retDate.Year(), retDate.Month(), retDate.Day(), opt.retHour, 40, 0, 0, time.UTC)
		retArrTime := retDepTime.Add(time.Duration(dur) * time.Minute)

		flights = append(flights, Flight{
			Price:               price,
			Airline:             opt.name,
			AirlineCode:         opt.code,
			FlightNumber:        opt.flightNum,
			DepartureTime:       depTime.Format(time.RFC3339),
			ArrivalTime:         arrTime.Format(time.RFC3339),
			Duration:            formatDurationMin(dur),
			Stops:               opt.stops,
			ReturnDepartureTime: retDepTime.Format(time.RFC3339),
			ReturnArrivalTime:   retArrTime.Format(time.RFC3339),
			ReturnDuration:      formatDurationMin(dur),
			ReturnStops:         opt.stops,
			Currency:            "USD",
		})
	}
	return flights
}

func estimateRoute(origin, destination string) routeData {
	type region struct{ lat, lon float64 }
	regions := map[byte]region{
		'A': {25, 55}, 'B': {30, 70}, 'C': {35, 105}, 'D': {50, 10},
		'E': {55, 25}, 'F': {10, 20}, 'G': {10, -15}, 'H': {30, 35},
		'J': {35, 130}, 'K': {37, 127}, 'L': {45, 15}, 'M': {40, 45},
		'N': {40, -75}, 'O': {60, 25}, 'P': {8, 125}, 'R': {55, 37},
		'S': {0, 110}, 'T': {40, 65}, 'U': {55, 65}, 'V': {20, 100},
		'W': {10, 120}, 'Y': {60, 15}, 'Z': {25, 120},
	}
	r1, r2 := regions[origin[0]], regions[destination[0]]
	if r1.lat == 0 { r1 = region{40, 40} }
	if r2.lat == 0 { r2 = region{40, 40} }

	dlat := (r2.lat - r1.lat) * math.Pi / 180
	dlon := (r2.lon - r1.lon) * math.Pi / 180
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(r1.lat*math.Pi/180)*math.Cos(r2.lat*math.Pi/180)*math.Sin(dlon/2)*math.Sin(dlon/2)
	distKm := 6371 * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	durationM := int(distKm/800*60) + 30
	if durationM < 60 { durationM = 60 }
	basePrice := int(distKm*0.12) + 80

	stops := 0
	if distKm > 5000 { stops = 1 }

	return routeData{
		basePrice: basePrice,
		durationM: durationM,
		airlines: []fallbackAirline{
			{"Turkish Airlines", "TK", "TK" + origin[:2] + "1", 1.00, stops, 7, 15},
			{"Emirates", "EK", "EK" + destination[:2] + "1", 1.20, stops, 9, 17},
			{"Qatar Airways", "QR", "QR" + origin[:2] + "2", 1.15, stops, 5, 14},
			{"Wizz Air", "W6", "W6" + origin[:2] + "3", 0.72, stops + 1, 4, 22},
			{"Lufthansa", "LH", "LH" + origin[:2] + "4", 1.10, stops, 11, 12},
		},
	}
}

// GenerateHotelsFallback produces realistic hotel data for major cities.
func GenerateHotelsFallback(destination string) []Hotel {
	cityHotels := map[string][]Hotel{
		"IST": {
			{"Grand Hyatt Istanbul", "", 189, 4.7, "Taksim, Istanbul", "", "USD"},
			{"Hilton Istanbul Bosphorus", "", 172, 4.5, "Beşiktaş, Istanbul", "", "USD"},
			{"The Marmara Taksim", "", 145, 4.4, "Taksim Square, Istanbul", "", "USD"},
			{"Sultan Ahmet Palace Hotel", "", 99, 4.3, "Sultanahmet, Istanbul", "", "USD"},
			{"ibis Istanbul Taksim", "", 72, 4.0, "Taksim, Istanbul", "", "USD"},
		},
		"CDG": {
			{"Hôtel Le Marais Bastille", "", 225, 4.6, "Le Marais, Paris", "", "USD"},
			{"Pullman Paris Tour Eiffel", "", 285, 4.5, "7th Arr., Paris", "", "USD"},
			{"Hôtel des Arts Montmartre", "", 135, 4.3, "Montmartre, Paris", "", "USD"},
			{"ibis Paris Opéra", "", 98, 4.0, "9th Arr., Paris", "", "USD"},
			{"Generator Paris", "", 58, 3.8, "10th Arr., Paris", "", "USD"},
		},
		"PAR": {
			{"Hôtel Le Marais Bastille", "", 225, 4.6, "Le Marais, Paris", "", "USD"},
			{"Pullman Paris Tour Eiffel", "", 285, 4.5, "7th Arr., Paris", "", "USD"},
			{"Hôtel des Arts Montmartre", "", 135, 4.3, "Montmartre, Paris", "", "USD"},
			{"ibis Paris Opéra", "", 98, 4.0, "9th Arr., Paris", "", "USD"},
			{"Generator Paris", "", 58, 3.8, "10th Arr., Paris", "", "USD"},
		},
		"LHR": {
			{"Hilton London Tower Bridge", "", 185, 4.4, "Tower Bridge, London", "", "USD"},
			{"The Hoxton Shoreditch", "", 168, 4.5, "Shoreditch, London", "", "USD"},
			{"citizenM London Bankside", "", 148, 4.4, "Bankside, London", "", "USD"},
			{"Premier Inn London City", "", 97, 4.1, "City of London", "", "USD"},
			{"Generator London", "", 52, 3.8, "Russell Square, London", "", "USD"},
		},
		"LON": {
			{"Hilton London Tower Bridge", "", 185, 4.4, "Tower Bridge, London", "", "USD"},
			{"The Hoxton Shoreditch", "", 168, 4.5, "Shoreditch, London", "", "USD"},
			{"citizenM London Bankside", "", 148, 4.4, "Bankside, London", "", "USD"},
			{"Premier Inn London City", "", 97, 4.1, "City of London", "", "USD"},
			{"Generator London", "", 52, 3.8, "Russell Square, London", "", "USD"},
		},
		"DXB": {
			{"JW Marriott Marquis Dubai", "", 228, 4.6, "Business Bay, Dubai", "", "USD"},
			{"Hilton Dubai Al Habtoor City", "", 165, 4.4, "Dubai Marina", "", "USD"},
			{"Atlantis The Palm", "", 390, 4.7, "Palm Jumeirah, Dubai", "", "USD"},
			{"Rove Downtown Dubai", "", 98, 4.3, "Downtown Dubai", "", "USD"},
			{"Premier Inn Dubai Ibn Battuta", "", 68, 4.0, "Jebel Ali, Dubai", "", "USD"},
		},
		"FRA": {
			{"Steigenberger Frankfurter Hof", "", 285, 4.6, "Kaiserplatz, Frankfurt", "", "USD"},
			{"Hilton Frankfurt City Centre", "", 178, 4.5, "City Centre, Frankfurt", "", "USD"},
			{"Marriott Frankfurt City Center", "", 158, 4.4, "Sachsenhausen, Frankfurt", "", "USD"},
			{"Motel One Frankfurt-Römer", "", 91, 4.3, "Römer, Frankfurt", "", "USD"},
			{"Generator Frankfurt", "", 48, 3.9, "Sachsenhausen, Frankfurt", "", "USD"},
		},
		"BER": {
			{"Hotel Adlon Kempinski", "", 325, 4.8, "Unter den Linden, Berlin", "", "USD"},
			{"Radisson Blu Berlin", "", 152, 4.4, "Alexanderplatz, Berlin", "", "USD"},
			{"Michelberger Hotel", "", 132, 4.5, "Friedrichshain, Berlin", "", "USD"},
			{"Motel One Berlin Hackescher Markt", "", 87, 4.2, "Mitte, Berlin", "", "USD"},
			{"Generator Berlin Mitte", "", 46, 3.9, "Mitte, Berlin", "", "USD"},
		},
		"JFK": {
			{"The Plaza Hotel", "", 590, 4.7, "Midtown, New York", "", "USD"},
			{"Marriott Marquis Times Square", "", 315, 4.5, "Times Square, New York", "", "USD"},
			{"citizenM New York Bowery", "", 189, 4.4, "Lower East Side, New York", "", "USD"},
			{"ibis New York Midtown", "", 148, 4.1, "Midtown, New York", "", "USD"},
			{"HI NYC Hostel", "", 65, 3.8, "Upper West Side, New York", "", "USD"},
		},
		"NYC": {
			{"The Plaza Hotel", "", 590, 4.7, "Midtown, New York", "", "USD"},
			{"Marriott Marquis Times Square", "", 315, 4.5, "Times Square, New York", "", "USD"},
			{"citizenM New York Bowery", "", 189, 4.4, "Lower East Side, New York", "", "USD"},
			{"ibis New York Midtown", "", 148, 4.1, "Midtown, New York", "", "USD"},
			{"HI NYC Hostel", "", 65, 3.8, "Upper West Side, New York", "", "USD"},
		},
		"BKK": {
			{"Mandarin Oriental Bangkok", "", 285, 4.8, "Charoennakorn, Bangkok", "", "USD"},
			{"Chatrium Hotel Riverside", "", 148, 4.5, "Riverside, Bangkok", "", "USD"},
			{"Novotel Bangkok Ploenchit", "", 118, 4.3, "Ploenchit, Bangkok", "", "USD"},
			{"ibis Bangkok Sukhumvit", "", 72, 4.2, "Sukhumvit, Bangkok", "", "USD"},
			{"Lub d Silom", "", 38, 4.0, "Silom, Bangkok", "", "USD"},
		},
		"SIN": {
			{"Marina Bay Sands", "", 485, 4.7, "Marina Bay, Singapore", "", "USD"},
			{"Fullerton Hotel Singapore", "", 368, 4.8, "Fullerton Square, Singapore", "", "USD"},
			{"ibis Singapore on Bencoolen", "", 112, 4.1, "Bencoolen, Singapore", "", "USD"},
			{"V Hotel Lavender", "", 88, 4.0, "Lavender, Singapore", "", "USD"},
			{"Wink Hostel", "", 42, 4.2, "Chinatown, Singapore", "", "USD"},
		},
		"NRT": {
			{"Park Hyatt Tokyo", "", 520, 4.8, "Shinjuku, Tokyo", "", "USD"},
			{"Shinjuku Granbell Hotel", "", 148, 4.4, "Shinjuku, Tokyo", "", "USD"},
			{"ibis Tokyo Shinjuku", "", 95, 4.1, "Shinjuku, Tokyo", "", "USD"},
			{"UNPLAN Shinjuku", "", 58, 4.3, "Shinjuku, Tokyo", "", "USD"},
			{"APA Hotel Shinjuku Kabukicho", "", 78, 4.0, "Kabukicho, Tokyo", "", "USD"},
		},
		"TYO": {
			{"Park Hyatt Tokyo", "", 520, 4.8, "Shinjuku, Tokyo", "", "USD"},
			{"Shinjuku Granbell Hotel", "", 148, 4.4, "Shinjuku, Tokyo", "", "USD"},
			{"ibis Tokyo Shinjuku", "", 95, 4.1, "Shinjuku, Tokyo", "", "USD"},
			{"UNPLAN Shinjuku", "", 58, 4.3, "Shinjuku, Tokyo", "", "USD"},
			{"APA Hotel Shinjuku Kabukicho", "", 78, 4.0, "Kabukicho, Tokyo", "", "USD"},
		},
		"MAD": {
			{"Hotel Ritz Madrid", "", 348, 4.8, "Paseo del Prado, Madrid", "", "USD"},
			{"NH Collection Madrid Gran Vía", "", 165, 4.5, "Gran Vía, Madrid", "", "USD"},
			{"Only YOU Hotel Atocha", "", 195, 4.6, "Atocha, Madrid", "", "USD"},
			{"ibis Madrid Centro", "", 82, 4.0, "Lavapiés, Madrid", "", "USD"},
			{"Generator Madrid", "", 48, 3.9, "Chueca, Madrid", "", "USD"},
		},
		"BCN": {
			{"Hotel Arts Barcelona", "", 385, 4.7, "Barceloneta, Barcelona", "", "USD"},
			{"Novotel Barcelona City", "", 158, 4.4, "Eixample, Barcelona", "", "USD"},
			{"Yurbban Passage Hotel", "", 135, 4.5, "El Born, Barcelona", "", "USD"},
			{"ibis Barcelona Centro", "", 85, 4.0, "Gothic Quarter, Barcelona", "", "USD"},
			{"Generator Barcelona", "", 46, 3.8, "Gràcia, Barcelona", "", "USD"},
		},
		"AMS": {
			{"Sofitel Legend The Grand Amsterdam", "", 398, 4.8, "Old Centre, Amsterdam", "", "USD"},
			{"Mövenpick Hotel Amsterdam City Centre", "", 168, 4.4, "Eastern Docklands, Amsterdam", "", "USD"},
			{"The Student Hotel Amsterdam City", "", 135, 4.3, "Amsterdam West", "", "USD"},
			{"ibis Amsterdam Centre", "", 105, 4.1, "De Wallen, Amsterdam", "", "USD"},
			{"Generator Amsterdam", "", 52, 3.9, "Oost, Amsterdam", "", "USD"},
		},
		"FCO": {
			{"Hotel de Russie", "", 425, 4.8, "Piazza del Popolo, Rome", "", "USD"},
			{"Colosseum Hotel", "", 128, 4.3, "Colosseo, Rome", "", "USD"},
			{"Bettoja Hotel Massimo D'Azeglio", "", 165, 4.4, "Termini, Rome", "", "USD"},
			{"ibis Roma Tiburtina", "", 78, 4.0, "Tiburtina, Rome", "", "USD"},
			{"Generator Rome", "", 44, 3.8, "Termini, Rome", "", "USD"},
		},
	}

	if hotels, ok := cityHotels[destination]; ok {
		return hotels
	}

	return []Hotel{
		{"Grand Hotel " + destination, "", 178, 4.5, "City Center, " + destination, "", "USD"},
		{"Marriott " + destination, "", 148, 4.4, "Business District, " + destination, "", "USD"},
		{"ibis " + destination + " Centre", "", 88, 4.1, "Central " + destination, "", "USD"},
		{"Boutique Residence " + destination, "", 122, 4.3, "Arts Quarter, " + destination, "", "USD"},
		{"Generator " + destination, "", 48, 3.8, "Student Quarter, " + destination, "", "USD"},
	}
}

// ─── Smart Built-in AI Summary ────────────────────────────────────────────────

func SmartFallbackRecommendation(budget float64, origin, destination, departureDate, returnDate string, passengers int, flights []Flight, hotels []Hotel) string {
	if len(flights) == 0 || len(hotels) == 0 {
		return "Unable to provide recommendations — no flight or hotel data available."
	}

	numNights := 3
	if dep, err := time.Parse("2006-01-02", departureDate); err == nil {
		if ret, err := time.Parse("2006-01-02", returnDate); err == nil {
			numNights = int(ret.Sub(dep).Hours() / 24)
		}
	}

	bestFlight := flights[0]
	cheapest := flights[0]
	premium := flights[0]
	for _, f := range flights {
		if f.Price < cheapest.Price { cheapest = f }
		if f.Price > premium.Price { premium = f }
		if f.Stops == 0 && f.Price < bestFlight.Price { bestFlight = f }
	}

	bestHotel := hotels[0]
	luxuryHotel := hotels[0]
	budgetHotel := hotels[0]
	for _, h := range hotels {
		if h.Price > luxuryHotel.Price { luxuryHotel = h }
		if h.Price < budgetHotel.Price { budgetHotel = h }
		if h.Rating/h.Price > bestHotel.Rating/bestHotel.Price { bestHotel = h }
	}

	totalBestValue := bestFlight.Price*float64(passengers) + bestHotel.Price*float64(numNights)
	totalBudget := cheapest.Price*float64(passengers) + budgetHotel.Price*float64(numNights)
	totalLuxury := premium.Price*float64(passengers) + luxuryHotel.Price*float64(numNights)

	budgetStatus := "within"
	if totalBestValue > budget { budgetStatus = "slightly over" }

	depFormatted := departureDate
	if t, err := time.Parse("2006-01-02", departureDate); err == nil {
		depFormatted = t.Format("Jan 2")
	}
	retFormatted := returnDate
	if t, err := time.Parse("2006-01-02", returnDate); err == nil {
		retFormatted = t.Format("Jan 2")
	}

	directLabel := "non-stop"
	if bestFlight.Stops > 0 { directLabel = fmt.Sprintf("%d-stop", bestFlight.Stops) }

	return fmt.Sprintf(
		"✈ Flight: **%s** at $%.0f/person — a %s flight (%s) offering the best balance of price and convenience for your %s→%s trip departing %s, returning %s.\n\n"+
			"🏨 Hotel: **%s** at $%.0f/night in %s (★%.1f) is your best value stay. With %d night(s) this adds $%.0f to your total.\n\n"+
			"💰 Budget Summary: Best-value combo comes to approximately **$%.0f** for %d passenger(s) — %s your $%.0f budget. "+
			"Budget option: %s + %s ≈ $%.0f. Premium option: %s + %s ≈ $%.0f.",
		bestFlight.Airline, bestFlight.Price,
		directLabel, bestFlight.Duration,
		origin, destination, depFormatted, retFormatted,
		bestHotel.Name, bestHotel.Price, bestHotel.Location, bestHotel.Rating,
		numNights, bestHotel.Price*float64(numNights),
		totalBestValue, passengers, budgetStatus, budget,
		cheapest.Airline, budgetHotel.Name, totalBudget,
		premium.Airline, luxuryHotel.Name, totalLuxury,
	)
}

// FallbackRecommendation kept for compatibility
func FallbackRecommendation(budget float64, flights []Flight, hotels []Hotel, numNights int) string {
	if len(flights) == 0 || len(hotels) == 0 {
		return "Unable to provide recommendations at this time."
	}
	cheapestFlight := flights[0]
	for _, f := range flights {
		if f.Price < cheapestFlight.Price { cheapestFlight = f }
	}
	bestValueHotel := hotels[0]
	for _, h := range hotels {
		if h.Price < bestValueHotel.Price { bestValueHotel = h }
	}
	total := cheapestFlight.Price + bestValueHotel.Price*float64(numNights)
	withinBudget := fmt.Sprintf(" Estimated total: $%.0f fits your $%.0f budget.", total, budget)
	if total > budget {
		withinBudget = fmt.Sprintf(" Note: $%.0f total exceeds your $%.0f budget by $%.0f.", total, budget, total-budget)
	}
	return fmt.Sprintf("Best picks: %s at $%.0f and %s at $%.0f/night (★%.1f).%s",
		cheapestFlight.Airline, cheapestFlight.Price,
		bestValueHotel.Name, bestValueHotel.Price, bestValueHotel.Rating,
		withinBudget)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func parseDuration(iso string) string {
	if iso == "" { return "" }
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
		if result != "" { result += " " }
		result += iso[:mIdx] + "m"
	}
	return result
}

func formatDurationMin(minutes int) string {
	h := minutes / 60
	m := minutes % 60
	if m > 0 { return fmt.Sprintf("%dh %dm", h, m) }
	return fmt.Sprintf("%dh", h)
}

func parsePrice(s string) float64 {
	var price float64
	fmt.Sscanf(s, "%f", &price)
	return price
}

func parseRating(s string) float64 {
	if s == "" { return 4.0 }
	var r float64
	fmt.Sscanf(s, "%f", &r)
	if r <= 0 { return 4.0 }
	if r > 5 { r = 5 }
	return r
}

func max(a, b int) int {
	if a > b { return a }
	return b
}

func airportToCity(airport string) string {
	mapping := map[string]string{
		"LHR": "LON", "LGW": "LON", "STN": "LON", "LTN": "LON",
		"CDG": "PAR", "ORY": "PAR",
		"JFK": "NYC", "LGA": "NYC", "EWR": "NYC",
		"LAX": "LAX", "DXB": "DXB", "IST": "IST", "FRA": "FRA",
		"AMS": "AMS", "BER": "BER", "SXF": "BER",
		"MAD": "MAD", "BCN": "BCN",
		"FCO": "ROM", "CIA": "ROM",
		"TAS": "TAS", "NRT": "TYO", "HND": "TYO",
		"SIN": "SIN", "BKK": "BKK",
	}
	if city, ok := mapping[airport]; ok { return city }
	return airport
}

func airlineName(code string) string {
	names := map[string]string{
		"TK": "Turkish Airlines", "LH": "Lufthansa", "AF": "Air France",
		"BA": "British Airways", "EK": "Emirates", "QR": "Qatar Airways",
		"PC": "Pegasus Airlines", "FR": "Ryanair", "U2": "EasyJet",
		"W6": "Wizz Air", "FZ": "FlyDubai", "HY": "Uzbekistan Airways",
		"UA": "United Airlines", "AA": "American Airlines", "DL": "Delta Air Lines",
		"KL": "KLM", "IB": "Iberia", "AZ": "ITA Airways",
		"OS": "Austrian Airlines", "LX": "Swiss International Air Lines",
		"SQ": "Singapore Airlines", "CX": "Cathay Pacific",
		"NH": "ANA", "JL": "Japan Airlines", "EY": "Etihad Airways",
		"SV": "Saudi Arabian Airlines", "MS": "EgyptAir", "RJ": "Royal Jordanian",
		"ET": "Ethiopian Airlines", "G9": "Air Arabia", "XQ": "SunExpress",
		"HV": "Transavia", "VY": "Vueling", "VS": "Virgin Atlantic",
		"TG": "Thai Airways", "N0": "Norse Atlantic", "TR": "Scoot",
	}
	if name, ok := names[code]; ok { return name }
	if code != "" { return code + " Airlines" }
	return "Unknown Airline"
}
