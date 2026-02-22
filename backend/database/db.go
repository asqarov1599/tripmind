package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// ─── Models ──────────────────────────────────────────────────────────────────

type Search struct {
	ID            string    `json:"id"`
	Origin        string    `json:"origin"`
	Destination   string    `json:"destination"`
	DepartureDate string    `json:"departure_date"`
	ReturnDate    string    `json:"return_date"`
	Budget        float64   `json:"budget"`
	Passengers    int       `json:"passengers"`
	CreatedAt     time.Time `json:"created_at"`
}

type Itinerary struct {
	ID           string    `json:"id"`
	SearchID     string    `json:"search_id"`
	FlightsJSON  string    `json:"flights_json"`
	HotelsJSON   string    `json:"hotels_json"`
	AISummary    string    `json:"ai_summary"`
	PDFData      []byte    `json:"pdf_data,omitempty"` // stored in DB, no filesystem needed
	TravelerName string    `json:"traveler_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func InitDB() {
	dsn := buildDSN()

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("❌ Failed to open database: %v", err)
	}

	// Connection pool settings suitable for Railway's free PostgreSQL
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Retry connection up to 10 times (Railway DB may take a moment to be ready)
	for i := 0; i < 10; i++ {
		if err = DB.Ping(); err == nil {
			break
		}
		log.Printf("⏳ Waiting for database... attempt %d/10: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("❌ Failed to connect to database after retries: %v", err)
	}

	migrate()
	log.Println("✅ Database connected and migrated")
}

func buildDSN() string {
	// Railway provides DATABASE_URL (postgres://user:pass@host:port/db)
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	// Fallback to individual vars (local dev)
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASSWORD", "postgres")
	name := getEnv("DB_NAME", "tripmind")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, sslmode)
}

// ─── Migrations ───────────────────────────────────────────────────────────────

func migrate() {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS searches (
			id            TEXT PRIMARY KEY,
			origin        TEXT NOT NULL,
			destination   TEXT NOT NULL,
			departure_date TEXT NOT NULL,
			return_date   TEXT NOT NULL,
			budget        NUMERIC(12,2) NOT NULL,
			passengers    INTEGER DEFAULT 1,
			created_at    TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS itineraries (
			id            TEXT PRIMARY KEY,
			search_id     TEXT NOT NULL REFERENCES searches(id),
			flights_json  TEXT,
			hotels_json   TEXT,
			ai_summary    TEXT,
			pdf_data      BYTEA,
			traveler_name TEXT,
			created_at    TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE INDEX IF NOT EXISTS idx_itineraries_search_id
			ON itineraries(search_id)`,

		`CREATE INDEX IF NOT EXISTS idx_searches_created_at
			ON searches(created_at DESC)`,
	}

	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			log.Fatalf("❌ Migration failed: %v\nSQL: %s", err, m)
		}
	}
}

// ─── CRUD ─────────────────────────────────────────────────────────────────────

func SaveSearch(s *Search) error {
	_, err := DB.Exec(`
		INSERT INTO searches (id, origin, destination, departure_date, return_date, budget, passengers)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.Origin, s.Destination, s.DepartureDate, s.ReturnDate, s.Budget, s.Passengers)
	return err
}

func GetSearch(id string) (*Search, error) {
	s := &Search{}
	err := DB.QueryRow(`
		SELECT id, origin, destination, departure_date, return_date, budget, passengers, created_at
		FROM searches WHERE id = $1`, id).
		Scan(&s.ID, &s.Origin, &s.Destination, &s.DepartureDate, &s.ReturnDate,
			&s.Budget, &s.Passengers, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func SaveItinerary(i *Itinerary) error {
	_, err := DB.Exec(`
		INSERT INTO itineraries (id, search_id, flights_json, hotels_json, ai_summary, pdf_data, traveler_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		i.ID, i.SearchID, i.FlightsJSON, i.HotelsJSON, i.AISummary, i.PDFData, i.TravelerName)
	return err
}

func UpdateItineraryPDF(id string, pdfData []byte, travelerName string) error {
	_, err := DB.Exec(`
		UPDATE itineraries SET pdf_data = $1, traveler_name = $2 WHERE id = $3`,
		pdfData, travelerName, id)
	return err
}

func GetItinerary(id string) (*Itinerary, error) {
	i := &Itinerary{}
	err := DB.QueryRow(`
		SELECT id, search_id, flights_json, hotels_json, ai_summary, pdf_data, traveler_name, created_at
		FROM itineraries WHERE id = $1`, id).
		Scan(&i.ID, &i.SearchID, &i.FlightsJSON, &i.HotelsJSON,
			&i.AISummary, &i.PDFData, &i.TravelerName, &i.CreatedAt)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func GetItineraryBySearchID(searchID string) (*Itinerary, error) {
	i := &Itinerary{}
	err := DB.QueryRow(`
		SELECT id, search_id, flights_json, hotels_json, ai_summary, pdf_data, traveler_name, created_at
		FROM itineraries WHERE search_id = $1
		ORDER BY created_at DESC LIMIT 1`, searchID).
		Scan(&i.ID, &i.SearchID, &i.FlightsJSON, &i.HotelsJSON,
			&i.AISummary, &i.PDFData, &i.TravelerName, &i.CreatedAt)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
