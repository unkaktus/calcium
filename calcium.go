//go:build linux

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	cpuid "github.com/klauspost/cpuid/v2"
)

type ExecTime struct {
	User   time.Duration
	System time.Duration
}

func GetExecTime() (*ExecTime, error) {
	tms := syscall.Tms{}
	_, err := syscall.Times(&tms)
	if err != nil {
		return nil, fmt.Errorf("syscall Times: %w", err)
	}
	execTime := &ExecTime{
		User:   time.Duration(float64(tms.Utime+tms.Cutime)*10) * time.Millisecond,
		System: time.Duration(float64(tms.Stime+tms.Cstime)*10) * time.Millisecond,
	}
	return execTime, nil
}

func RunTransparentCommand() error {
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command: %w", err)
	}
	return nil
}

func WriteReport() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home directory: %w", err)
	}

	calciumDir := path.Join(homeDir, ".calcium")
	if err := os.MkdirAll(calciumDir, 0755); err != nil {
		return fmt.Errorf("create calcium directory: %w", err)
	}
	reportFilename := filepath.Join(calciumDir, "calcium-report.csv")
	reportFile, err := os.OpenFile(reportFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0775)
	if err != nil {
		return fmt.Errorf("open report file: %w", err)
	}
	defer reportFile.Close()

	execTime, err := GetExecTime()
	if err != nil {
		return err
	}

	binaryName := filepath.Base(os.Args[1])

	report := strings.Join([]string{
		time.Now().Format(time.DateTime),
		"\"" + cpuid.CPU.BrandName + "\"",
		binaryName,
		fmt.Sprintf("%.2f", execTime.User.Seconds()),
		fmt.Sprintf("%.2f", execTime.System.Seconds()),
	}, ",")

	_, err = fmt.Fprintf(reportFile, "%s\n", report)
	if err != nil {
		return fmt.Errorf("write report to file: %w", err)
	}
	return nil
}

func run() error {
	RunTransparentCommand()
	if err := WriteReport(); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
