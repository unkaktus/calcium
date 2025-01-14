package calcium

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	partialparser "github.com/blaze2305/partial-json-parser"
	"github.com/blaze2305/partial-json-parser/options"
	"golang.org/x/net/html"
)

func getVendorDomain(cpuString string) string {
	if strings.HasPrefix(cpuString, "Intel") {
		return "www.intel.com"
	}
	if strings.HasPrefix(cpuString, "AMD") {
		return "www.amd.com"
	}
	return ""
}

func buildQuery(cpuString string) (string, error) {
	vendorDomain := getVendorDomain(cpuString)
	if vendorDomain == "" {
		return "", fmt.Errorf("unknown vendor")
	}
	query := "! " + cpuString + " site:" + vendorDomain
	if vendorDomain == "www.amd.com" {
		query = "! " + cpuString + " drivers and support site:" + vendorDomain
	}
	return query, nil
}

func GetSpecPageURL(cpuString string) (string, error) {
	query, err := buildQuery(cpuString)
	if err != nil {
		return "", fmt.Errorf("build query: %w", err)
	}
	q := url.QueryEscape(query)
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
	return location.String(), nil
}

type AMDSpecs struct {
	Elements struct {
		DefaultTDP struct {
			FormatValue string `json:"formatValue"`
		} `json:"defaultTdp"`
		NumOfCpuCores struct {
			FormatValue string `json:"formatValue"`
		} `json:"numOfCpuCores"`
	} `json:"elements"`
}

func tokenHasAttributeValue(token html.Token, key, value string) bool {
	for _, attr := range token.Attr {
		if attr.Key == key {
			sp := strings.Split(attr.Val, " ")
			for _, s := range sp {
				if s == value {
					return true
				}
			}
		}
	}
	return false
}

func iterateUntilAttribute(z *html.Tokenizer, key, value string) error {
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				break
			}
			return z.Err()
		}
		if tokenHasAttributeValue(z.Token(), key, value) {
			return nil
		}
	}
	return io.EOF
}

func skipTokens(z *html.Tokenizer, n int) error {
	for i := 0; i < n; i++ {
		tt := z.Next()
		if tt == html.ErrorToken {
			return z.Err()
		}
	}
	return nil
}

func ExtractTDP(specURL string) (float64, error) {
	req, _ := http.NewRequest(http.MethodGet, specURL, http.NoBody)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return 0, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	z := html.NewTokenizer(resp.Body)

	TotalTDP := 0.0
	CoreCount := 0.0

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				break
			}
			return 0, z.Err()
		}
		token := z.Token()

		if token.Data == "div" {
			if tokenHasAttributeValue(token, "class", "tech-section-row") {
				if err := iterateUntilAttribute(z, "class", "tech-label"); err != nil {
					return 0, err
				}
				if err := skipTokens(z, 3); err != nil {
					return 0, err
				}
				raw := z.Raw()
				field := strings.TrimSpace(string(raw))
				if !slices.Contains([]string{"TDP", "Total Cores"}, field) {
					continue
				}

				if err := iterateUntilAttribute(z, "class", "tech-data"); err != nil {
					return 0, err
				}
				if err := skipTokens(z, 3); err != nil {
					return 0, err
				}
				raw = z.Raw()
				s := strings.TrimSpace(string(raw))

				switch field {
				case "TDP":
					if !strings.HasSuffix(s, " W") {
						continue
					}
					s = strings.TrimRight(s, " W")
					tdp, err := strconv.ParseFloat(s, 32)
					if err != nil {
						break
					}
					TotalTDP = tdp
				case "Total Cores":
					coreCount, err := strconv.ParseFloat(s, 32)
					if err != nil {
						break
					}
					CoreCount = coreCount
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
					coreCount, err := strconv.ParseFloat(specs.Elements.NumOfCpuCores.FormatValue, 32)
					if err != nil {
						break
					}
					TotalTDP = tdp
					CoreCount = coreCount
				}
			}
		}
	}

	if TotalTDP == 0 || CoreCount == 0 {
		return 0, fmt.Errorf("TDP not found")
	}

	tdp := TotalTDP / CoreCount
	return tdp, nil
}

type TDPInfo struct {
	CPUString string
	Watts     float64
	Source    string
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
		CPUString: cpuString,
		Watts:     tdp,
		Source:    specURL,
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
			CPUString: cpuString,
			Watts:     tdp,
			Source:    source,
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
		fmt.Sprintf("%.4f", tdpInfo.Watts),
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
