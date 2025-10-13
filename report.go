package calcium

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/unkaktus/calcium/data"
)

// Return regional carbon intesity per unit of energy [gCO2e/kWh]
func GetCarbonIntensityRegion(region string) (*data.CarbonIntensity, error) {
	carbonIntensity, ok := data.CarbonIntensities[region]
	if !ok {
		return nil, fmt.Errorf("unknown region")
	}
	return &carbonIntensity, nil
}

type Consumption struct {
	CPUTime float64 // [h]
	Energy  float64 // [kWh]
	CO2e    float64 `json:",omitempty"` // [kg]
}

type Report struct {
	Timestamp           string
	Software            string
	Region              string `json:",omitempty"`
	CarbonIntensityYear int    `json:",omitempty"`
	Units               map[string]string
	Tags                map[string]*Consumption
}

func MakeReport(logFilename, region string, nodeFactor float64) error {
	if logFilename == "" {
		calciumDir, err := getCalciumDir()
		if err != nil {
			return fmt.Errorf("get calcium directory: %w", err)
		}
		logFilename = filepath.Join(calciumDir, "log.csv")
	}
	logFile, err := os.OpenFile(logFilename, os.O_RDONLY, 0775)
	if err != nil {
		return fmt.Errorf("open report file: %w", err)
	}
	defer logFile.Close()

	csvReader := csv.NewReader(logFile)
	records, err := csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("read log file: %w", err)
	}

	report := Report{
		Software:  "github.com/unkaktus/calcium",
		Timestamp: time.Now().Format(time.DateTime),
		Tags:      map[string]*Consumption{},
		Units: map[string]string{
			"CPUTime": "h",
			"Energy":  "kWh",
			"CO2e":    "kg",
		},
	}

	var carbonIntensity *data.CarbonIntensity
	if region != "none" {
		carbonIntensity, err = GetCarbonIntensityRegion(region)
		if err != nil {
			return fmt.Errorf("get emissions per energy unit: %w", err)
		}
		report.Region = region
		report.CarbonIntensityYear = carbonIntensity.Year
	}

	for _, row := range records {
		if len(row) < 5 {
			return fmt.Errorf("invalid row length")
		}
		tag := row[2]

		if _, ok := report.Tags[tag]; !ok {
			report.Tags[tag] = &Consumption{}
		}

		// Sum the CPU times up
		userCPUTime, err := strconv.ParseFloat(row[3], 32)
		if err != nil {
			return fmt.Errorf("parse user CPU time: %w", err)
		}
		systemCPUTime, err := strconv.ParseFloat(row[4], 32)
		if err != nil {
			return fmt.Errorf("parse user CPU time: %w", err)
		}
		localCPUTime := (userCPUTime + systemCPUTime) / 3600 // In hours
		report.Tags[tag].CPUTime += localCPUTime

		// Calculate energy
		cpuString := row[1]
		tdpInfo, err := GetTDPInfoCached(cpuString)
		if err != nil {
			return fmt.Errorf("get TDP info: %w", err)
		}
		localEnergy := localCPUTime * (tdpInfo.Watts * 1e-3) * nodeFactor
		report.Tags[tag].Energy += localEnergy

		// Calculate CO2e
		if region != "none" {
			report.Tags[tag].CO2e += localEnergy * (1e-3 * carbonIntensity.Value)
		}
	}

	jsonData, _ := json.MarshalIndent(report, "", "     ")
	fmt.Printf("%s\n", jsonData)

	return nil
}
