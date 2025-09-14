package config

import "os"

var IsProduction = os.Getenv("APP_ENV") == "production"
