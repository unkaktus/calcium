## 🪨 calcium

`calcium` is a command wrapper tool to collect its CPU usage and report the estimated carbon footprint.

[![DOI](https://zenodo.org/badge/829550127.svg)](https://doi.org/10.5281/zenodo.13876575)
[![Go Reference](https://pkg.go.dev/badge/github.com/unkaktus/calcium.svg)](https://pkg.go.dev/github.com/unkaktus/calcium)

### Installation

1. Download the latest binary for your architecture at the [releases page](https://github.com/unkaktus/calcium/releases).
2. Copy to the executable path, e.g., to `$HOME/bin`.
3. Add or make sure that the executable path is added to `PATH` variable:
```shell
export PATH=$PATH:$HOME/bin
```

### Compiling from source

0. Install Go (https://go.dev)

1. Build `calcium` for Linux:
```shell
go install github.com/unkaktus/calcium/cmd/calcium@latest
export PATH=$PATH:$HOME/go/bin
```

### Usage
## Collection

Run any app transparently with a project tag:

```shell
calcium run -tag Project1337 ./analyze data.dat
```

It will then output to `$HOME/.calcium/log.csv` the following information in CSV format:

```
Timestamp, CPU Name, Tag, User CPU Time [s], System CPU Time [s]
```

For example,

```
2024-09-20 19:50:49,"Intel(R) Xeon(R) Platinum 8270 CPU @ 2.70GHz",Project1337,0.48,0.61
```

Tag value is recommended to be unique and traceable to a specific workload, such as job name or ID.

## Reporting
Once your runs are done, it's time to obtain the emission footprint report.

To create a report specify the ISO 3-letter country code (ISO 3166-1 alpha-3) and run
```shell
calcium report -region DEU
```

The output will be in JSON format, e.g.,
```json
{
  "Timestamp": "2024-09-22 17:45:04",
  "Software": "github.com/unkaktus/calcium",
  "Region": "DEU",
  "CarbonIntensityYear": 2023,
  "Units": {
    "CO2e": "kg",
    "CPUTime": "s",
    "Energy": "kWh"
  },
  "Tags": {
    "NSbh": {
      "CPUTime": 29699999744,
      "Energy": 60156.5235436358,
      "CO2e": 22916.655917514123
    },
  }
}
```

The carbon intenity data is provided by Ember, Energy Institute, and Our World in Data.

> Ember (2024); Energy Institute - Statistical Review of World Energy (2024) –
> with major processing by Our World in Data.
> “Carbon intensity of electricity generation – Ember and Energy Institute” [dataset].
> Ember, “Yearly Electricity Data”; Energy Institute, “Statistical Review of World Energy” [original data].
> Retrieved September 22, 2024 from https://ourworldindata.org/grapher/carbon-intensity-electricity
