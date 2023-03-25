package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	OrgId            string
	Regions          Regions
	Zones            Zones
	ExportProjectId  string
	ExportBucketName string
}

func NewConfig() *Config {
	return &Config{
		OrgId:            "",
		Regions:          nil,
		Zones:            nil,
		ExportProjectId:  "",
		ExportBucketName: "",
	}
}

var DefaultConfig = &Config{
	OrgId:            os.Getenv("ORG_ID"),
	Regions:          getStringListFromEnv("REGIONS"),
	Zones:            getStringListFromEnv("ZONES"),
	ExportProjectId:  os.Getenv("EXPORT_PROJECT_ID"),
	ExportBucketName: os.Getenv("EXPORT_BUCKET_NAME"),
}

func getStringListFromEnv(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}

func (c *Config) Validate() error {
	if c.OrgId == "" {
		return fmt.Errorf("ORG_ID is missing")
	}
	if len(c.Regions) == 0 {
		return fmt.Errorf("REGIONS is missing")
	}
	if len(c.Zones) == 0 {
		return fmt.Errorf("ZONES is missing")
	}
	if c.ExportProjectId == "" {
		return fmt.Errorf("EXPORT_PROJECT_ID is missing")
	}
	if c.ExportBucketName == "" {
		return fmt.Errorf("EXPORT_BUCKET_NAME is missing")
	}
	return nil
}
