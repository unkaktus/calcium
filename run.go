package calcium

import (
	"fmt"
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

func RunTransparentCommand(cmdline []string) error {
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command: %w", err)
	}
	return nil
}

func getCalciumDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home directory: %w", err)
	}

	calciumDir := path.Join(homeDir, ".calcium")
	if err := os.MkdirAll(calciumDir, 0755); err != nil {
		return "", fmt.Errorf("create calcium directory: %w", err)
	}
	return calciumDir, nil
}

func WriteLog(tag string) error {
	calciumDir, err := getCalciumDir()
	if err != nil {
		return fmt.Errorf("get calcium directory: %w", err)
	}

	logFilename := filepath.Join(calciumDir, "log.csv")
	logFile, err := os.OpenFile(logFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0775)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	execTime, err := GetExecTime()
	if err != nil {
		return err
	}

	log := strings.Join([]string{
		time.Now().Format(time.DateTime),
		"\"" + cpuid.CPU.BrandName + "\"",
		tag,
		fmt.Sprintf("%.2f", execTime.User.Seconds()),
		fmt.Sprintf("%.2f", execTime.System.Seconds()),
	}, ",")

	_, err = fmt.Fprintf(logFile, "%s\n", log)
	if err != nil {
		return fmt.Errorf("write log to file: %w", err)
	}
	return nil
}
