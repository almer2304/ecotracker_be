package utils

import "math"

const earthRadiusKm = 6371.0

// HaversineDistance menghitung jarak antara dua titik koordinat dalam kilometer
// menggunakan formula Haversine
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Konversi derajat ke radian
	lat1Rad := toRad(lat1)
	lat2Rad := toRad(lat2)
	deltaLat := toRad(lat2 - lat1)
	deltaLon := toRad(lon2 - lon1)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// CollectorWithDistance menggabungkan collector dengan jaraknya
type CollectorWithDistance struct {
	ID         string
	Lat        float64
	Lon        float64
	DistanceKm float64
}

// SortByDistance mengurutkan slice CollectorWithDistance dari jarak terdekat
func SortByDistance(collectors []CollectorWithDistance) {
	// Simple insertion sort untuk slice kecil (biasanya < 100 collector)
	for i := 1; i < len(collectors); i++ {
		key := collectors[i]
		j := i - 1
		for j >= 0 && collectors[j].DistanceKm > key.DistanceKm {
			collectors[j+1] = collectors[j]
			j--
		}
		collectors[j+1] = key
	}
}
