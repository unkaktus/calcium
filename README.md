## 🪨 calcium

`calcium` is a wrapper tool to collect the CPU usage history, making possible later estimations of the carbon footprint.

### Manual building

0. Install Go (https://go.dev)

1. Build `calcium` for Linux:
```shell
go install github.com/unkaktus/calcium@latest
export PATH=$PATH:$HOME/go/bin
```

### Usage

Run any app transparently:

```shell
calcium ./invert_matrix data.dat
```

It will then output to `$HOME/.calcium/calcium-report.csv` the following information in CSV format:

```
Timestamp, CPU Name, Binary Name, User CPU Time [s], System CPU Time [s]
```

For example,

```
2024-07-16 23:43:04,"Intel(R) Xeon(R) Gold 6248R CPU @ 3.00GHz",htop,0.08,0.15
```