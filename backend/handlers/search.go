package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
	"tripmind/database"
	"tripmind/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SearchRequest struct {
	Origin        string  `json:"origin" binding:"required"`
	Destination   string  `json:"destination" binding:"required"`
	DepartureDate string  `json:"departure_date" binding:"required"`
	ReturnDate    string  `json:"return_date" binding:"required"`
	Budget        float64 `json:"budget" binding:"required,gt=0"`
	Passengers    int     `json:"passengers"`
}

type SearchResponse struct {
	SearchID  string           `json:"search_id"`
	Flights   []services.Flight `json:"flights"`
	Hotels    []services.Hotel  `json:"hotels"`
	AISummary string           `json:"ai_summary"`
	Source    string           `json:"source"` // "live" or "estimated"
}

func SearchHandler(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	req.Origin = strings.ToUpper(strings.TrimSpace(req.Origin))
	req.Destination = strings.ToUpper(strings.TrimSpace(req.Destination))

	if req.Passengers <= 0 {
		req.Passengers = 1
	}

	// Validate airport code length
	if len(req.Origin) != 3 || len(req.Destination) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Airport codes must be exactly 3 characters (e.g. LHR, JFK)"})
		return
	}

	// Validate dates
	depDate, err := time.Parse("2006-01-02", req.DepartureDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid departure date format. Use YYYY-MM-DD"})
		return
	}

	retDate, err := time.Parse("2006-01-02", req.ReturnDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid return date format. Use YYYY-MM-DD"})
		return
	}

	if !retDate.After(depDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Return date must be after departure date"})
		return
	}

	numNights := int(retDate.Sub(depDate).Hours() / 24)

	// ── Try Amadeus live data ──────────────────────────────────────────────────
	var flights []services.Flight
	var hotels []services.Hotel
	isFallback := false
	source := "live"

	amadeusClient := services.GetAmadeusClient()

	// Flights
	if amadeusClient != nil {
		liveFlights, err := amadeusClient.SearchFlights(
			req.Origin, req.Destination,
			req.DepartureDate, req.ReturnDate,
			req.Passengers,
		)
		if err != nil {
			log.Printf("⚠️  Amadeus flight search failed: %v — using fallback", err)
			flights = services.GenerateFlightsFallback(req.Origin, req.Destination, req.DepartureDate, req.ReturnDate)
			isFallback = true
		} else if len(liveFlights) == 0 {
			log.Println("⚠️  Amadeus returned 0 flights — using fallback")
			flights = services.GenerateFlightsFallback(req.Origin, req.Destination, req.DepartureDate, req.ReturnDate)
			isFallback = true
		} else {
			flights = liveFlights
			log.Printf("✅ Amadeus: %d live flights found", len(flights))
		}
	} else {
		flights = services.GenerateFlightsFallback(req.Origin, req.Destination, req.DepartureDate, req.ReturnDate)
		isFallback = true
	}

	// Hotels
	if amadeusClient != nil && !isFallback {
		liveHotels, err := amadeusClient.SearchHotels(
			req.Destination,
			req.DepartureDate,
			req.ReturnDate,
			req.Passengers,
		)
		if err != nil {
			log.Printf("⚠️  Amadeus hotel search failed: %v — using fallback", err)
			hotels = services.GenerateHotelsFallback(req.Destination)
			isFallback = true
		} else if len(liveHotels) == 0 {
			log.Println("⚠️  Amadeus returned 0 hotels — using fallback")
			hotels = services.GenerateHotelsFallback(req.Destination)
			isFallback = true
		} else {
			hotels = liveHotels
			log.Printf("✅ Amadeus: %d live hotels found", len(hotels))
		}
	} else {
		if hotels == nil {
			hotels = services.GenerateHotelsFallback(req.Destination)
		}
		isFallback = true
	}

	if isFallback {
		source = "estimated"
	}

	// ── AI Recommendations ────────────────────────────────────────────────────
	aiClient := services.GetAIClient()
	aiSummary, err := aiClient.GetRecommendations(
		req.Budget, req.Origin, req.Destination,
		req.DepartureDate, req.ReturnDate,
		req.Passengers, flights, hotels, isFallback,
	)
	if err != nil {
		log.Printf("⚠️  AI recommendation failed: %v — using fallback text", err)
		aiSummary = services.FallbackRecommendation(req.Budget, flights, hotels, numNights)
	}

	// ── Persist to DB ─────────────────────────────────────────────────────────
	searchID := uuid.New().String()
	if err := database.SaveSearch(&database.Search{
		ID:            searchID,
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureDate: req.DepartureDate,
		ReturnDate:    req.ReturnDate,
		Budget:        req.Budget,
		Passengers:    req.Passengers,
	}); err != nil {
		log.Printf("❌ Failed to save search: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save search"})
		return
	}

	flightsJSON, _ := json.Marshal(flights)
	hotelsJSON, _ := json.Marshal(hotels)

	itineraryID := uuid.New().String()
	if err := database.SaveItinerary(&database.Itinerary{
		ID:          itineraryID,
		SearchID:    searchID,
		FlightsJSON: string(flightsJSON),
		HotelsJSON:  string(hotelsJSON),
		AISummary:   aiSummary,
	}); err != nil {
		log.Printf("❌ Failed to save itinerary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save itinerary"})
		return
	}

	c.JSON(http.StatusOK, SearchResponse{
		SearchID:  searchID,
		Flights:   flights,
		Hotels:    hotels,
		AISummary: aiSummary,
		Source:    source,
	})
}
