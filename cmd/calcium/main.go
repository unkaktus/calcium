package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/klauspost/cpuid/v2"
	"github.com/minio/selfupdate"
	"github.com/unkaktus/calcium"
	"github.com/urfave/cli/v2"
)

func run() error {
	app := &cli.App{
		Name:     "calcium",
		HelpName: "calcium",
		Usage:    "Tracking energy consumption of computing workloads",
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Ivan Markin",
				Email: "git@unkaktus.art",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Transparently run the given application",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "tag",
						Usage: "Log consumption under this tag",
					},
				},
				Action: func(cCtx *cli.Context) error {
					cmdline := append([]string{cCtx.Args().First()}, cCtx.Args().Tail()...)

					tag := cCtx.String("tag")

					if tag == "" {
						binaryName := filepath.Base(cmdline[0])
						tag = binaryName
					}

					if err := calcium.RunTransparentCommand(cmdline); err != nil {
						return fmt.Errorf("run command: %w", err)
					}
					if err := calcium.WriteLog(tag); err != nil {
						return fmt.Errorf("write log: %w", err)
					}

					return nil
				},
			},
			{
				Name:  "tdp",
				Usage: "Get TDP of a CPU by its CPUID string",
				Action: func(cCtx *cli.Context) error {
					cpuString := cCtx.Args().Get(0)
					if cpuString == "" {
						cpuString = cpuid.CPU.BrandName
					}

					tdpInfo, err := calcium.GetTDPInfoCached(cpuString)
					if err != nil {
						return fmt.Errorf("get TDP info: %w", err)
					}
					jsonData, _ := json.Marshal(tdpInfo)
					fmt.Printf("%s\n", jsonData)

					return nil
				},
			},
			{
				Name:  "report",
				Usage: "Report on the aggregated consumption",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "region",
						Usage:    "Region to calculate emission cost of electricity",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "logfile",
						Usage: "Filename of the log file",
					},
				},
				Action: func(cCtx *cli.Context) error {
					region := cCtx.String("region")
					logFilename := cCtx.String("logfile")
					err := calcium.MakeReport(logFilename, region)
					return err
				},
			},
			{
				Name:  "update",
				Usage: "update itself",
				Action: func(cCtx *cli.Context) error {
					calciumURL := fmt.Sprintf("https://github.com/unkaktus/calcium/releases/latest/download/calcium-%s-%s", runtime.GOOS, runtime.GOARCH)
					resp, err := http.Get(calciumURL)
					if err != nil {
						return fmt.Errorf("download release binary: %w", err)
					}
					if resp.StatusCode != http.StatusOK {
						return fmt.Errorf("unsuccessful download: status %s", resp.Status)
					}
					fmt.Printf("Downloaded new binary.\n")
					defer resp.Body.Close()
					err = selfupdate.Apply(resp.Body, selfupdate.Options{})
					if err != nil {
						return fmt.Errorf("apply update: %w", err)
					}
					fmt.Printf("Successfully applied the update.\n")
					return nil
				},
			},
		},
	}
	return app.Run(os.Args)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
