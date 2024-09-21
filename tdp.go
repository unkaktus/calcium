package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	partialparser "github.com/blaze2305/partial-json-parser"
	"github.com/blaze2305/partial-json-parser/options"
	"golang.org/x/net/html"
)

func GetSpecPageURL(cpuString string) (string, error) {
	q := url.QueryEscape("! " + cpuString)
	u := "https://api.duckduckgo.com/?q=" + q + "&format=json"
	req, err := http.NewRequest(http.MethodGet, u, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", fmt.Errorf("get: %w", err)
	}
	location, err := resp.Location()
	if err != nil {
		return "", fmt.Errorf("get location: %w", err)
	}
	switch location.Host {
	case "ark.intel.com", "www.amd.com":
		return location.String(), nil
	}
	return "", fmt.Errorf("invalid cpu string")
}

type AMDSpecs struct {
	Elements struct {
		DefaultTDP struct {
			FormatValue string `json:"formatValue"`
		} `json:"defaultTdp"`
	} `json:"elements"`
}

func ExtractTDP(specURL string) (float64, error) {
	req, _ := http.NewRequest(http.MethodGet, specURL, http.NoBody)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return 0, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	r := resp.Body
	z := html.NewTokenizer(r)

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				break
			}
			return 0, z.Err()
		}
		token := z.Token()

		if token.Data == "span" {
			for _, attr := range token.Attr {
				if attr.Key == "data-key" && attr.Val == "MaxTDP" {
					z.Next()
					raw := z.Raw()
					s := strings.TrimSpace(string(raw))
					if !strings.HasSuffix(s, " W") {
						break
					}
					s = strings.TrimRight(s, " W")
					tdp, err := strconv.ParseFloat(s, 32)
					if err != nil {
						break
					}
					return tdp, nil
				}

			}
		}

		if token.Data == "div" {
			for _, attr := range token.Attr {
				if attr.Key == "data-product-specs" {
					d := html.UnescapeString(attr.Val)

					value, err := partialparser.ParseMalformedString(d, options.NUM|options.ARR|options.OBJ, false)
					if err != nil {
						return 0, fmt.Errorf("decode AMD specs: %w", err)
					}

					specs := &AMDSpecs{}
					err = json.Unmarshal([]byte(value), specs)
					if err != nil {
						return 0, fmt.Errorf("decode AMD specs: %w", err)
					}
					s := strings.TrimRight(specs.Elements.DefaultTDP.FormatValue, "W")
					tdp, err := strconv.ParseFloat(s, 32)
					if err != nil {
						break
					}
					return tdp, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("TDP not found")
}

type TDPInfo struct {
	Watts  float64
	Source string
}

func GetTDPInfo(cpuString string) (*TDPInfo, error) {
	specURL, err := GetSpecPageURL(cpuString)
	if err != nil {
		return nil, fmt.Errorf("get spec page: %w", err)
	}

	tdp, err := ExtractTDP(specURL)
	if err != nil {
		return nil, fmt.Errorf("get TDP: %w", err)
	}

	ti := &TDPInfo{
		Watts:  tdp,
		Source: specURL,
	}
	return ti, nil
}

func readTDPCache() (map[string]TDPInfo, error) {
	calciumDir, err := getCalciumDir()
	if err != nil {
		return nil, fmt.Errorf("get calcium directory: %w", err)
	}
	cacheFilename := filepath.Join(calciumDir, "tdp-cache.csv")
	cacheFile, err := os.OpenFile(cacheFilename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0775)
	if err != nil {
		return nil, fmt.Errorf("open report file: %w", err)
	}
	defer cacheFile.Close()

	csvReader := csv.NewReader(cacheFile)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read cache file: %w", err)
	}
	cache := map[string]TDPInfo{}
	for _, row := range records {
		if len(row) != 3 {
			return nil, fmt.Errorf("invalid TDP record length")
		}
		tdp, err := strconv.ParseFloat(row[1], 32)
		if err != nil {
			return nil, fmt.Errorf("parse TDP value: %w", err)
		}
		cpuString := row[0]
		source := row[2]
		cache[cpuString] = TDPInfo{
			Watts:  tdp,
			Source: source,
		}
	}
	return cache, nil
}

func writeTDPCache(cpuString string, tdpInfo TDPInfo) error {
	calciumDir, err := getCalciumDir()
	if err != nil {
		return fmt.Errorf("get calcium directory: %w", err)
	}
	cacheFilename := filepath.Join(calciumDir, "tdp-cache.csv")
	cacheFile, err := os.OpenFile(cacheFilename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0775)
	if err != nil {
		return fmt.Errorf("open report file: %w", err)
	}
	defer cacheFile.Close()

	entry := strings.Join([]string{
		"\"" + cpuString + "\"",
		fmt.Sprintf("%.2f", tdpInfo.Watts),
		tdpInfo.Source,
	}, ",")

	_, err = fmt.Fprintf(cacheFile, "%s\n", entry)
	if err != nil {
		return fmt.Errorf("write report to file: %w", err)
	}
	return nil
}

func GetTDPInfoCached(cpuString string) (*TDPInfo, error) {
	cache, err := readTDPCache()
	if err != nil {
		return nil, fmt.Errorf("read cache: %w", err)
	}

	if tdpInfo, ok := cache[cpuString]; ok {
		return &tdpInfo, nil
	}

	tdpInfo, err := GetTDPInfo(cpuString)
	if err != nil {
		return nil, err
	}

	if err := writeTDPCache(cpuString, *tdpInfo); err != nil {
		return nil, fmt.Errorf("write TDP cache: %w", err)
	}

	return tdpInfo, nil
}
