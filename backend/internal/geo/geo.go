package geo

import (
	"errors"
	"fmt"
	"seer/internal/utils"

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

	if !utils.ValidateIp(ip) {
		return nil, ErrInvalidIp
	}

	city, err := gs.db.Get_city(ip)
	if err != nil || city.City == "-" {
		return nil, fmt.Errorf("failed to get city: %w", err)
	}

	country, err := gs.db.Get_country_short(ip)
	if err != nil || country.Country_short == "-" {
		return nil, fmt.Errorf("failed to get country: %w", err)
	}

	return &GeoData{
		Country: country.Country_short,
		City:    city.City,
	}, nil
}
