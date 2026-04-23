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

	search, err := database.GetSearch(req.SearchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Search session not found"})
		return
	}

	itinerary, err := database.GetItineraryBySearchID(req.SearchID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Itinerary data not found"})
		return
	}

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

	passengers := search.Passengers
	if passengers <= 0 {
		passengers = 1
	}

	// Total = (flight price per person × passengers) + (hotel per night × nights)
	// Flight price from Amadeus is already the full round-trip price per person.
	totalCost := selectedFlight.Price*float64(passengers) + selectedHotel.Price*float64(numNights)

	pdfData := services.PDFData{
		TravelerName:  req.TravelerName,
		Origin:        search.Origin,
		Destination:   search.Destination,
		DepartureDate: search.DepartureDate,
		ReturnDate:    search.ReturnDate,
		Flight:        selectedFlight,
		Hotel:         selectedHotel,
		NumNights:     numNights,
		Passengers:    passengers,
		TotalCost:     totalCost,
		AISummary:     itinerary.AISummary,
	}

	pdfBytes, err := services.GeneratePDFBytes(pdfData)
	if err != nil {
		log.Printf("❌ PDF generation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

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