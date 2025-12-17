package kantine

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"time"
)

type TopLevelElement struct {
	SpeiseplanGerichtData []GerichtData `json:"speiseplanGerichtData"`
}

type GerichtData struct {
	SpeiseplanAdvancedGericht AdvancedGericht `json:"speiseplanAdvancedGericht"`
}

type AdvancedGericht struct {
	Datum       string `json:"datum"`       // Expected format: "YYYY-MM-DDTHH:MM:SS"
	Gerichtname string `json:"gerichtname"` // The name of the meal
}

type OutputMenu struct {
	Datum    string   `json:"datum"`    // Formatted date: "DD.MM.YYYY"
	Gerichte []string `json:"gerichte"` // List of meal names for that date
}

// --- Constants ---
const (
	apiURL      = "https://orangecampus-api-mt.konkaapps.de/kms-mt-webservices/webresources/entity.speiseplanadvanced/getdata/5500/1"
	apiUser     = "ws_live_user"
	apiPassword = "jTzLzWC.vjsqD4te" // Consider using environment variables or a config file for credentials in production
	// Time layout for parsing the input date string (matches "YYYY-MM-DDTHH:MM:SS")
	inputTimeLayout = "2006-01-02T15:04:05"
	// Time layout for formatting the output date string (matches "DD.MM.YYYY")
	outputTimeLayout = "02.01.2006"
)

func Fetch() ([]OutputMenu, error) {
	// 1. Fetch Data with Basic Auth
	client := &http.Client{
		Timeout: 30 * time.Second, // Set a reasonable timeout
	}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.SetBasicAuth(apiUser, apiPassword)
	req.Header.Set("Accept", "application/json") // Good practice to specify accept header

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Read body for error context
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// 2. Parse Initial JSON
	var topLevelData []TopLevelElement
	err = json.Unmarshal(bodyBytes, &topLevelData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling initial JSON: %v", err)
	}

	// 3. Flatten the data (equivalent to '.[] .speiseplanGerichtData')
	allGerichteData := []GerichtData{}
	for _, element := range topLevelData {
		allGerichteData = append(allGerichteData, element.SpeiseplanGerichtData...)
	}

	if len(allGerichteData) == 0 {
		return []OutputMenu{}, nil
	}

	// 4. Group by Date (equivalent to 'group_by(.speiseplanAdvancedGericht.datum)')
	// We use a map where the key is the original date string
	groupedByDate := make(map[string][]AdvancedGericht)
	dateKeys := []string{} // Keep track of dates to sort later

	for _, gericht := range allGerichteData {
		dateStr := gericht.SpeiseplanAdvancedGericht.Datum
		if _, exists := groupedByDate[dateStr]; !exists {
			groupedByDate[dateStr] = []AdvancedGericht{}
			dateKeys = append(dateKeys, dateStr) // Add new date key
		}
		groupedByDate[dateStr] = append(groupedByDate[dateStr], gericht.SpeiseplanAdvancedGericht)
	}

	// 5. Sort the dates chronologically before processing
	// This ensures the final output array is ordered by date
	sort.Slice(dateKeys, func(i, j int) bool {
		// Handle potential parse errors during sorting comparison, though they shouldn't occur if data is valid
		t1, _ := time.Parse(inputTimeLayout, dateKeys[i])
		t2, _ := time.Parse(inputTimeLayout, dateKeys[j])
		return t1.Before(t2)
	})

	// 6. Transform/Map Data and Format Output
	finalOutput := []OutputMenu{}

	for _, dateStr := range dateKeys { // Iterate using sorted keys
		gerichteGroup := groupedByDate[dateStr]

		if len(gerichteGroup) == 0 {
			continue // Should not happen based on grouping logic, but safe check
		}

		// Parse the date string (use the key itself)
		parsedTime, err := time.Parse(inputTimeLayout, dateStr)
		if err != nil {
			slog.Warn("Skipping date due to parse error", "date", dateStr, "error", err)
			continue // Skip this group if date parsing fails
		}
		formattedDate := parsedTime.Format(outputTimeLayout)

		// Extract meal names (equivalent to 'map(.speiseplanAdvancedGericht.gerichtname)')
		mealNames := make([]string, 0, len(gerichteGroup))
		for _, gericht := range gerichteGroup {
			mealNames = append(mealNames, gericht.Gerichtname)
		}

		// Add the processed group to the final output
		finalOutput = append(finalOutput, OutputMenu{
			Datum:    formattedDate,
			Gerichte: mealNames,
		})
	}
	return finalOutput, nil
}
