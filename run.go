package calcium

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	cpuid "github.com/klauspost/cpuid/v2"
	"github.com/unkaktus/calcium/gpu/nvidia"
)

const killTimeout = 5 * time.Second

func StartTransparentCommand(cmdline []string) (*exec.Cmd, error) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	done := make(chan bool, 1)
	defer func() {
		done <- true
	}()

	go func(cmd *exec.Cmd) {
		sig := <-signals
		cmd.Process.Signal(sig)
		select {
		case <-time.After(killTimeout):
		case <-done:
		}
		cmd.Process.Kill()
	}(cmd)

	return cmd, nil
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

	if err := syscall.Flock(int(logFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire log file lock: %w", err)
	}
	defer syscall.Flock(int(logFile.Fd()), syscall.LOCK_UN)

	cpuTime, err := GetCPUTime()
	if err != nil {
		return err
	}

	log := strings.Join([]string{
		time.Now().Format(time.DateTime),
		"\"" + cpuid.CPU.BrandName + "\"",
		tag,
		fmt.Sprintf("%.2f", cpuTime.User.Seconds()),
		fmt.Sprintf("%.2f", cpuTime.System.Seconds()),
	}, ",")

	_, err = fmt.Fprintf(logFile, "%s\n", log)
	if err != nil {
		return fmt.Errorf("write log to file: %w", err)
	}
	return nil
}

func WriteGPULog(gpuPoller GPUEnergyPoller, tag string) error {
	if gpuPoller.TotalEnergy() == 0 {
		return nil
	}
	calciumDir, err := getCalciumDir()
	if err != nil {
		return fmt.Errorf("get calcium directory: %w", err)
	}

	logFilename := filepath.Join(calciumDir, "log-gpu.csv")
	logFile, err := os.OpenFile(logFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0775)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	if err := syscall.Flock(int(logFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire log file lock: %w", err)
	}
	defer syscall.Flock(int(logFile.Fd()), syscall.LOCK_UN)

	log := strings.Join([]string{
		time.Now().Format(time.DateTime),
		tag,
		fmt.Sprintf("\"%s\"", gpuPoller.DeviceName()),
		fmt.Sprintf("\"%s\"", strings.Join(gpuPoller.UsedDevices(), ",")),
		fmt.Sprintf("%e", gpuPoller.TotalEnergy()),
	}, ",")

	_, err = fmt.Fprintf(logFile, "%s\n", log)
	if err != nil {
		return fmt.Errorf("write log to file: %w", err)
	}
	return nil
}

func Run(cmdline []string, tag string) error {
	if tag == "" {
		binaryName := filepath.Base(cmdline[0])
		tag = binaryName
	}

	// Always write usage log
	defer func() {
		if err := WriteLog(tag); err != nil {
			log.Printf("write log: %v", err)
		}
	}()

	cmd, err := StartTransparentCommand(cmdline)
	if err != nil {
		return fmt.Errorf("run command: %w", err)
	}

	var gpuPoller GPUEnergyPoller
	interval := time.Second

	gpuPoller, err = nvidia.NewEnergyPoller(interval, uint32(cmd.Process.Pid))
	if err == nil {
		defer func() {
			if err := gpuPoller.Stop(); err != nil {
				log.Printf("stop GPU energy poller: %w", err)
			}
		}()

		defer func() {
			if err := WriteGPULog(gpuPoller, tag); err != nil {
				log.Printf("write log: %v", err)
			}
		}()
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
