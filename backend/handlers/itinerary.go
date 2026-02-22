package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"tripmind/database"
	"tripmind/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GenerateRequest struct {
	SearchID            string `json:"search_id" binding:"required"`
	SelectedFlightIndex int    `json:"selected_flight_index"`
	SelectedHotelIndex  int    `json:"selected_hotel_index"`
	TravelerName        string `json:"traveler_name"`
}

type GenerateResponse struct {
	ItineraryID string `json:"itinerary_id"`
	PDFURL      string `json:"pdf_url"`
	Message     string `json:"message"`
}

func GenerateHandler(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Fetch search from DB
	search, err := database.GetSearch(req.SearchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Search session not found"})
		return
	}

	// Fetch cached itinerary data
	itinerary, err := database.GetItineraryBySearchID(req.SearchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Itinerary data not found"})
		return
	}

	// Parse flights + hotels from cached JSON
	var flights []services.Flight
	var hotels []services.Hotel

	if err := json.Unmarshal([]byte(itinerary.FlightsJSON), &flights); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse cached flight data"})
		return
	}
	if err := json.Unmarshal([]byte(itinerary.HotelsJSON), &hotels); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse cached hotel data"})
		return
	}

	// Bounds-check the selected indices
	if req.SelectedFlightIndex < 0 || req.SelectedFlightIndex >= len(flights) {
		req.SelectedFlightIndex = 0
	}
	if req.SelectedHotelIndex < 0 || req.SelectedHotelIndex >= len(hotels) {
		req.SelectedHotelIndex = 0
	}

	selectedFlight := flights[req.SelectedFlightIndex]
	selectedHotel := hotels[req.SelectedHotelIndex]

	depDate, _ := time.Parse("2006-01-02", search.DepartureDate)
	retDate, _ := time.Parse("2006-01-02", search.ReturnDate)
	numNights := int(retDate.Sub(depDate).Hours() / 24)
	totalCost := selectedFlight.Price + selectedHotel.Price*float64(numNights)

	// Generate PDF bytes (no filesystem — stored in PostgreSQL)
	pdfData := services.PDFData{
		TravelerName:  req.TravelerName,
		Origin:        search.Origin,
		Destination:   search.Destination,
		DepartureDate: search.DepartureDate,
		ReturnDate:    search.ReturnDate,
		Flight:        selectedFlight,
		Hotel:         selectedHotel,
		NumNights:     numNights,
		TotalCost:     totalCost,
		AISummary:     itinerary.AISummary,
	}

	pdfBytes, err := services.GeneratePDFBytes(pdfData)
	if err != nil {
		log.Printf("❌ PDF generation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	// Save new itinerary record with PDF bytes
	newID := uuid.New().String()
	newItin := &database.Itinerary{
		ID:           newID,
		SearchID:     req.SearchID,
		FlightsJSON:  itinerary.FlightsJSON,
		HotelsJSON:   itinerary.HotelsJSON,
		AISummary:    itinerary.AISummary,
		PDFData:      pdfBytes,
		TravelerName: req.TravelerName,
	}

	if err := database.SaveItinerary(newItin); err != nil {
		log.Printf("❌ Failed to save itinerary with PDF: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save generated PDF"})
		return
	}

	log.Printf("✅ PDF generated for itinerary %s (%d bytes)", newID, len(pdfBytes))

	c.JSON(http.StatusOK, GenerateResponse{
		ItineraryID: newID,
		PDFURL:      "/api/download/" + newID,
		Message:     "PDF generated successfully",
	})
}
