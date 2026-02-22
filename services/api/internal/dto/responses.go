package dto

import "time"

// CustomerUsage represents current usage vs plan ceilings with percentages.
type CustomerUsage struct {
	CPUCores    float64   `json:"cpuCores"`
	CPUCeiling  int       `json:"cpuCeiling"`
	CPUPercent  float64   `json:"cpuPercent"`
	RAMGB       float64   `json:"ramGb"`
	RAMCeiling  int       `json:"ramCeiling"`
	RAMPercent  float64   `json:"ramPercent"`
	S3TB        float64   `json:"s3Tb"`
	S3Ceiling   int       `json:"s3Ceiling"`
	S3Percent   float64   `json:"s3Percent"`
	DBStorageGB float64   `json:"dbStorageGb"`
	DBCeiling   int       `json:"dbCeiling"`
	DBPercent   float64   `json:"dbPercent"`
	VolumeGB    float64   `json:"volumeGb"`
	VolCeiling  int       `json:"volCeiling"`
	VolPercent  float64   `json:"volPercent"`
	LBCount     int       `json:"lbCount"`
	LBCeiling   int       `json:"lbCeiling"`
	LBPercent   float64   `json:"lbPercent"`
	RecordedAt  time.Time `json:"recordedAt"`
}

// UsageHistoryEntry represents a daily aggregated usage record.
type UsageHistoryEntry struct {
	Date        string  `json:"date"`
	CPUAvg      float64 `json:"cpuAvg"`
	CPUMax      float64 `json:"cpuMax"`
	RAMAvg      float64 `json:"ramAvg"`
	RAMMax      float64 `json:"ramMax"`
	DBStorageGB float64 `json:"dbStorageGb"`
	VolumeGB    float64 `json:"volumeGb"`
	LBCount     int     `json:"lbCount"`
}

// PlatformUsageSummary aggregates usage across all customers.
type PlatformUsageSummary struct {
	TotalCPU           float64 `json:"totalCpu"`
	TotalRAM           float64 `json:"totalRam"`
	TotalStorage       float64 `json:"totalStorage"`
	CustomersReporting int     `json:"customersReporting"`
}
