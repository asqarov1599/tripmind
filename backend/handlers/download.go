package handlers

import (
	"net/http"
	"tripmind/database"

	"github.com/gin-gonic/gin"
)

func DownloadHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing itinerary ID"})
		return
	}

	itinerary, err := database.GetItinerary(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Itinerary not found"})
		return
	}

	if len(itinerary.PDFData) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "PDF has not been generated for this itinerary"})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=tripmind-itinerary.pdf")
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/pdf", itinerary.PDFData)
}

func HealthHandler(c *gin.Context) {
	db := database.DB
	dbStatus := "ok"
	if db == nil {
		dbStatus = "not initialized"
	} else if err := db.Ping(); err != nil {
		dbStatus = "error: " + err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"service":  "TripMind API",
		"database": dbStatus,
	})
}
