package geo

import (
	"errors"
	"fmt"

	"github.com/ip2location/ip2location-go/v9"
)

type GeoService struct {
	db *ip2location.DB
}

var ErrInvalidIp = errors.New("invalid ip")

type GeoData struct {
	Country string
	City    string
}

func NewGeoService(db *ip2location.DB) *GeoService {
	return &GeoService{
		db: db,
	}
}

func (gs *GeoService) GetGeoDataFromIp(ip string) (*GeoData, error) {

	if !ip2location.OpenTools().IsIPv4(ip) && !ip2location.OpenTools().IsIPv6(ip) {
		return nil, ErrInvalidIp
	}

	city, err := gs.db.Get_city(ip)
	if err != nil {
		return nil, fmt.Errorf("failed to get city: %w", err)
	}

	country, err := gs.db.Get_country_short(ip)
	if err != nil {
		return nil, fmt.Errorf("failed to get country: %w", err)
	}

	return &GeoData{
		Country: country.Country_short,
		City:    city.City,
	}, nil
}
