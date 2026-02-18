package utils

import (
	"math"
	"sort"
)

const earthRadiusKm = 6371.0

// CalculateDistance returns the distance in kilometers between two GPS coordinates
// using the Haversine formula
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)

	lat1Rad := degreesToRadians(lat1)
	lat2Rad := degreesToRadians(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// PickupWithDistance wraps pickup data with calculated distance
type PickupWithDistance struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	PhotoURL    string  `json:"photo_url"`
	Status      string  `json:"status"`
	DistanceKm  float64 `json:"distance_km"`
	CreatedAt   string  `json:"created_at"`
}

// SortPickupsByDistance sorts a list of pickups by distance from a given location
func SortPickupsByDistance(pickups interface{}, fromLat, fromLon float64) []PickupWithDistance {
	// This is a generic helper - in real implementation, you'd pass a typed slice
	// For now, this shows the concept
	var result []PickupWithDistance
	return result
}

// CalculateDistancesAndSort takes pickups and collector location, returns sorted by distance
func CalculateDistancesAndSort(pickups []map[string]interface{}, collectorLat, collectorLon float64) []PickupWithDistance {
	var result []PickupWithDistance

	for _, p := range pickups {
		pickupLat := p["latitude"].(float64)
		pickupLon := p["longitude"].(float64)

		distance := CalculateDistance(collectorLat, collectorLon, pickupLat, pickupLon)

		result = append(result, PickupWithDistance{
			ID:         p["id"].(string),
			UserID:     p["user_id"].(string),
			Address:    p["address"].(string),
			Latitude:   pickupLat,
			Longitude:  pickupLon,
			PhotoURL:   p["photo_url"].(string),
			Status:     p["status"].(string),
			DistanceKm: distance,
			CreatedAt:  p["created_at"].(string),
		})
	}

	// Sort by distance (ascending - closest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].DistanceKm < result[j].DistanceKm
	})

	return result
}
