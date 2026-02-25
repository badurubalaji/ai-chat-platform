package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Patient represents a patient record
type Patient struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	DOB       string    `json:"dob,omitempty"`
	Gender    string    `json:"gender,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Email     string    `json:"email,omitempty"`
	Address   string    `json:"address,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// HistoryEntry represents a patient history record
type HistoryEntry struct {
	ID        string    `json:"id"`
	PatientID string    `json:"patient_id"`
	Type      string    `json:"type"` // visit, lab, prescription, diagnosis
	Date      string    `json:"date"`
	Provider  string    `json:"provider"`
	Summary   string    `json:"summary"`
	Details   string    `json:"details,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	patients = sync.Map{}
	history  = sync.Map{} // patientID -> []HistoryEntry
)

func init() {
	// Seed some demo data
	seedPatients := []Patient{
		{ID: "P-1001", Name: "John Smith", DOB: "1985-03-15", Gender: "male", Phone: "555-0101", Status: "active", CreatedAt: time.Now()},
		{ID: "P-1002", Name: "Sarah Johnson", DOB: "1990-07-22", Gender: "female", Phone: "555-0102", Status: "active", CreatedAt: time.Now()},
		{ID: "P-1003", Name: "Michael Brown", DOB: "1978-11-08", Gender: "male", Phone: "555-0103", Status: "active", CreatedAt: time.Now()},
	}
	for _, p := range seedPatients {
		patients.Store(p.ID, p)
	}

	// Seed history for P-1001
	history.Store("P-1001", []HistoryEntry{
		{ID: "H-001", PatientID: "P-1001", Type: "visit", Date: "2025-12-10", Provider: "Dr. Williams", Summary: "Annual checkup - all vitals normal", Details: "BP: 120/80, HR: 72, Temp: 98.6F"},
		{ID: "H-002", PatientID: "P-1001", Type: "lab", Date: "2025-12-10", Provider: "Quest Diagnostics", Summary: "Blood panel - cholesterol slightly elevated", Details: "Total cholesterol: 215, LDL: 140, HDL: 55, Triglycerides: 100"},
		{ID: "H-003", PatientID: "P-1001", Type: "prescription", Date: "2025-12-15", Provider: "Dr. Williams", Summary: "Atorvastatin 10mg daily for cholesterol management"},
		{ID: "H-004", PatientID: "P-1001", Type: "visit", Date: "2026-01-20", Provider: "Dr. Williams", Summary: "Follow-up - cholesterol improving with medication", Details: "BP: 118/76, patient reports no side effects from statin"},
	})

	history.Store("P-1002", []HistoryEntry{
		{ID: "H-005", PatientID: "P-1002", Type: "visit", Date: "2026-01-05", Provider: "Dr. Chen", Summary: "Reported persistent headaches for 2 weeks"},
		{ID: "H-006", PatientID: "P-1002", Type: "diagnosis", Date: "2026-01-05", Provider: "Dr. Chen", Summary: "Tension-type headache, likely stress-related"},
		{ID: "H-007", PatientID: "P-1002", Type: "prescription", Date: "2026-01-05", Provider: "Dr. Chen", Summary: "Ibuprofen 400mg as needed, stress management referral"},
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/patients", handlePatients)
	mux.HandleFunc("/patients/", handlePatientByID)
	mux.HandleFunc("/patients/search", handleSearchPatients)
	mux.HandleFunc("/history/", handleHistory)

	port := "8085"
	log.Printf("Mock EHR API running on http://localhost:%s", port)
	log.Printf("Seeded 3 patients with history data")
	log.Fatal(http.ListenAndServe(":"+port, withLogging(withCORS(mux))))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID, X-User-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[EHR] %s %s (tenant: %s, user: %s)", r.Method, r.URL.Path, r.Header.Get("X-Tenant-ID"), r.Header.Get("X-User-ID"))
		next.ServeHTTP(w, r)
	})
}

func handlePatients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// List all patients
		var list []Patient
		patients.Range(func(key, value interface{}) bool {
			list = append(list, value.(Patient))
			return true
		})
		jsonResp(w, http.StatusOK, list)

	case "POST":
		// Create patient
		var req struct {
			Name    string `json:"name"`
			DOB     string `json:"dob"`
			Gender  string `json:"gender"`
			Phone   string `json:"phone"`
			Email   string `json:"email"`
			Address string `json:"address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" {
			jsonError(w, http.StatusBadRequest, "name is required")
			return
		}

		patient := Patient{
			ID:        fmt.Sprintf("P-%s", uuid.New().String()[:4]),
			Name:      req.Name,
			DOB:       req.DOB,
			Gender:    req.Gender,
			Phone:     req.Phone,
			Email:     req.Email,
			Address:   req.Address,
			Status:    "active",
			CreatedAt: time.Now(),
		}
		patients.Store(patient.ID, patient)
		log.Printf("[EHR] Created patient: %s (%s)", patient.ID, patient.Name)
		jsonResp(w, http.StatusCreated, patient)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePatientByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from /patients/{id}
	id := strings.TrimPrefix(r.URL.Path, "/patients/")
	if id == "" || id == "search" {
		// /patients/search is handled separately
		handleSearchPatients(w, r)
		return
	}

	val, ok := patients.Load(id)
	if !ok {
		jsonError(w, http.StatusNotFound, fmt.Sprintf("patient %s not found", id))
		return
	}

	jsonResp(w, http.StatusOK, val.(Patient))
}

func handleSearchPatients(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.ToLower(r.URL.Query().Get("name"))
	}
	if query == "" {
		jsonError(w, http.StatusBadRequest, "search query (q or name) is required")
		return
	}

	var results []Patient
	patients.Range(func(key, value interface{}) bool {
		p := value.(Patient)
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.ID), query) {
			results = append(results, p)
		}
		return true
	})

	jsonResp(w, http.StatusOK, results)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	// /history/{patient_id}
	patientID := strings.TrimPrefix(r.URL.Path, "/history/")
	if patientID == "" {
		jsonError(w, http.StatusBadRequest, "patient_id is required")
		return
	}

	// Check patient exists
	if _, ok := patients.Load(patientID); !ok {
		jsonError(w, http.StatusNotFound, fmt.Sprintf("patient %s not found", patientID))
		return
	}

	val, ok := history.Load(patientID)
	if !ok {
		jsonResp(w, http.StatusOK, []HistoryEntry{})
		return
	}

	jsonResp(w, http.StatusOK, val.([]HistoryEntry))
}

func jsonResp(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonResp(w, status, map[string]string{"error": msg})
}
