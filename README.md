## 🪨 calcium

`calcium` is a command wrapper tool to collect its CPU usage and report the estimated carbon footprint.

[![DOI](https://zenodo.org/badge/829550127.svg)](https://doi.org/10.5281/zenodo.13876575)
[![Go Reference](https://pkg.go.dev/badge/github.com/unkaktus/calcium.svg)](https://pkg.go.dev/github.com/unkaktus/calcium)

## Installation

1. Download the latest binary for your architecture (see [releases page](https://github.com/unkaktus/calcium/releases) for the binary name), and put it into `$HOME/bin`.
Here is the example for Linux on AMD64 architecture:

```shell
mkdir -p ~/bin
curl -L -o ~/bin/calcium https://github.com/unkaktus/calcium/releases/latest/download/calcium-linux-amd64
chmod +x ~/bin/calcium
```

2. Add or make sure that the executable path is added to `PATH` variable, as well as to your shell's rc file:
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

## Usage
### Collection

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

### Reporting
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
    "CPUTime": "h",
    "Energy": "kWh"
  },
  "Tags": {
    "NSbh": {
      "CPUTime": 8249999.928888889,
      "Energy": 60156.5235436358,
      "CO2e": 22916.655917514123
    },
  }
}
```

You can also obtain the TDP value for a given CPU ID string in JSON format:

```shell
calcium tdp "Intel Xeon Gold 6242"
```

## Citing and sources

The required citation is for the Zenodo code record:

```
@software{ivan_markin_2024_13876575,
  author       = {Ivan Markin},
  title        = {calcium - Tracking carbon footprint of computing},
  month        = oct,
  year         = 2024,
  publisher    = {Zenodo},
  version      = {v1.4.1},
  doi          = {10.5281/zenodo.13876575},
  url          = {https://doi.org/10.5281/zenodo.13876575}
}
```

The carbon intenity data is provided by Ember, Energy Institute, and Our World in Data.

> Ember (2024); Energy Institute - Statistical Review of World Energy (2024) –
> with major processing by Our World in Data.
> “Carbon intensity of electricity generation – Ember and Energy Institute” [dataset].
> Ember, “Yearly Electricity Data”; Energy Institute, “Statistical Review of World Energy” [original data].
> Retrieved September 22, 2024 from https://ourworldindata.org/grapher/carbon-intensity-electricity
