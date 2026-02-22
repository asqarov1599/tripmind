package services

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type PDFData struct {
	TravelerName  string
	Origin        string
	Destination   string
	DepartureDate string
	ReturnDate    string
	Flight        Flight
	Hotel         Hotel
	NumNights     int
	TotalCost     float64
	AISummary     string
	IsEstimated   bool // true when Amadeus is not configured
}

// GeneratePDFBytes generates a PDF and returns raw bytes (no filesystem needed)
func GeneratePDFBytes(data PDFData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// ── Watermark ────────────────────────────────────────────
	pdf.SetTextColor(230, 230, 230)
	pdf.SetFont("Helvetica", "B", 55)
	pdf.TransformBegin()
	pdf.TransformRotate(42, 60, 200)
	pdf.Text(60, 200, "SAMPLE")
	pdf.TransformEnd()
	pdf.SetTextColor(0, 0, 0)

	// ── Header Bar ───────────────────────────────────────────
	pdf.SetFillColor(13, 24, 37) // --navy-950
	pdf.Rect(0, 0, 210, 28, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(20, 8)
	pdf.CellFormat(100, 10, "TripMind", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(212, 168, 67) // gold
	pdf.SetXY(20, 18)
	pdf.CellFormat(170, 6, "AI-Powered Travel Itinerary", "", 1, "L", false, 0, "")

	pdf.SetY(35)
	pdf.SetTextColor(0, 0, 0)

	// ── Disclaimer ───────────────────────────────────────────
	pdf.SetFillColor(255, 248, 225)
	pdf.SetDrawColor(212, 168, 67)
	pdf.SetTextColor(130, 90, 20)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetLineWidth(0.4)
	y := pdf.GetY()
	pdf.Rect(20, y, 170, 12, "FD")
	pdf.SetXY(23, y+2)
	disclaimer := "⚠ This is NOT a booking confirmation. Prices are estimates and subject to change. Please verify with providers before booking."
	if data.IsEstimated {
		disclaimer = "⚠ ESTIMATED PRICES — Amadeus API not configured. This is NOT a booking confirmation. Verify all prices before booking."
	}
	pdf.MultiCell(164, 4, disclaimer, "", "C", false)

	pdf.SetTextColor(0, 0, 0)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.2)
	pdf.Ln(6)

	// ── Section Helper ───────────────────────────────────────
	sectionHeader := func(title string) {
		pdf.SetFillColor(13, 24, 37)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(170, 8, "  "+title, "", 1, "L", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(2)
	}

	row := func(label, value string) {
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(100, 100, 100)
		pdf.CellFormat(55, 7, label, "", 0, "L", false, 0, "")
		pdf.SetTextColor(20, 20, 20)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(115, 7, value, "", 1, "L", false, 0, "")
	}

	// ── Traveler Info ─────────────────────────────────────────
	sectionHeader("Traveler Information")
	name := data.TravelerName
	if name == "" {
		name = "Guest Traveler"
	}
	row("Name", name)
	row("Generated", time.Now().Format("02 Jan 2006, 15:04 UTC"))
	pdf.Ln(4)

	// ── Trip Overview ─────────────────────────────────────────
	sectionHeader("Trip Overview")
	row("Route", fmt.Sprintf("%s → %s → %s", data.Origin, data.Destination, data.Origin))
	row("Departure", fmtDateReadable(data.DepartureDate))
	row("Return", fmtDateReadable(data.ReturnDate))
	row("Duration", fmt.Sprintf("%d nights", data.NumNights))
	pdf.Ln(4)

	// ── Selected Flight ───────────────────────────────────────
	sectionHeader("Selected Flight")
	row("Airline", data.Flight.Airline)
	row("Outbound", formatFlightLeg(data.Flight.DepartureTime, data.Flight.ArrivalTime, data.Flight.Duration))
	row("Return", formatFlightLeg(data.Flight.ReturnDepartureTime, data.Flight.ReturnArrivalTime, data.Flight.ReturnDuration))
	stops := "Direct"
	if data.Flight.Stops > 0 {
		stops = fmt.Sprintf("%d stop(s)", data.Flight.Stops)
	}
	row("Stops", stops)
	row("Price", fmt.Sprintf("$%.0f per person (round-trip)", data.Flight.Price))
	pdf.Ln(4)

	// ── Selected Hotel ────────────────────────────────────────
	sectionHeader("Selected Hotel")
	row("Hotel", data.Hotel.Name)
	row("Location", data.Hotel.Location)
	row("Rating", fmt.Sprintf("%.1f / 5.0", data.Hotel.Rating))
	row("Check-in", fmtDateReadable(data.DepartureDate))
	row("Check-out", fmtDateReadable(data.ReturnDate))
	row("Price", fmt.Sprintf("$%.0f/night × %d nights = $%.0f",
		data.Hotel.Price, data.NumNights, data.Hotel.Price*float64(data.NumNights)))
	pdf.Ln(4)

	// ── Cost Summary ──────────────────────────────────────────
	sectionHeader("Cost Estimate")
	row("Flight (per person)", fmt.Sprintf("$%.0f", data.Flight.Price))
	row("Hotel total", fmt.Sprintf("$%.0f", data.Hotel.Price*float64(data.NumNights)))

	pdf.SetFillColor(212, 168, 67)
	pdf.SetTextColor(13, 24, 37)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(55, 9, "TOTAL ESTIMATE", "", 0, "L", true, 0, "")
	pdf.CellFormat(115, 9, fmt.Sprintf("$%.0f", data.TotalCost), "", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	// ── AI Summary ────────────────────────────────────────────
	if data.AISummary != "" {
		sectionHeader("AI Recommendations")
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(40, 40, 40)
		pdf.MultiCell(170, 5, data.AISummary, "", "L", false)
		pdf.Ln(4)
	}

	// ── Footer ────────────────────────────────────────────────
	pdf.SetY(-22)
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.3)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 8,
		"Generated by TripMind AI Travel Planner · Not a booking confirmation · Prices subject to change",
		"", 0, "C", false, 0, "")

	// ── Write to buffer ───────────────────────────────────────
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("PDF output failed: %w", err)
	}
	return buf.Bytes(), nil
}

func fmtDateReadable(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("02 Jan 2006 (Mon)")
}

func formatFlightLeg(dep, arr, dur string) string {
	depT, err1 := time.Parse(time.RFC3339, dep)
	arrT, err2 := time.Parse(time.RFC3339, arr)
	if err1 != nil || err2 != nil {
		if dep != "" && arr != "" {
			return dep + " → " + arr
		}
		return "N/A"
	}
	result := fmt.Sprintf("%s → %s",
		depT.Format("02 Jan 15:04"),
		arrT.Format("02 Jan 15:04"))
	if dur != "" {
		result += fmt.Sprintf(" (%s)", dur)
	}
	return result
}
