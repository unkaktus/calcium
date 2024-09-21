## 🪨 calcium

`calcium` is a wrapper tool to collect the CPU usage history, making possible later estimations of the carbon footprint.

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