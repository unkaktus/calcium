package nvidia

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

func getHomogeneousName() (string, error) {
	deviceCount, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return "", fmt.Errorf("get device count: %s", nvml.ErrorString(ret))
	}
	deviceName := ""
	for i := 0; i < deviceCount; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return "", fmt.Errorf("get device at index %d: %s", i, nvml.ErrorString(ret))
		}
		name, ret := device.GetName()
		if ret != nvml.SUCCESS {
			return "", fmt.Errorf("get name of the device %d: %s", i, nvml.ErrorString(ret))
		}
		if deviceName == "" {
			deviceName = name
			continue
		}
		if name != deviceName {
			return "", fmt.Errorf("devices do not have the same names")
		}
	}
	return deviceName, nil
}

type EnergyPoller struct {
	interval    time.Duration
	pid         uint32
	deviceCount int
	deviceName  string
	usedDevices map[int]struct{}
	ticker      *time.Ticker
	mutex       sync.RWMutex

	totalEnergy float64 // kWh
}

func NewEnergyPoller(interval time.Duration, pid uint32) (*EnergyPoller, error) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("Unable to initialize NVML: %s", nvml.ErrorString(ret))
	}

	p := &EnergyPoller{
		interval:    interval,
		pid:         pid,
		usedDevices: map[int]struct{}{},
	}

	deviceCount, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("get device count: %s", nvml.ErrorString(ret))
	}
	p.deviceCount = deviceCount

	deviceName, err := getHomogeneousName()
	if err != nil {
		return nil, fmt.Errorf("get homogeneous name: %w", err)
	}
	p.deviceName = deviceName

	p.ticker = time.NewTicker(p.interval)

	go func() {
		for range p.ticker.C {
			if err := p.poll(); err != nil {
				log.Printf("EnergyPoller poll: %v", err)
			}
		}
	}()

	return p, nil
}

func (p *EnergyPoller) poll() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i := 0; i < p.deviceCount; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return fmt.Errorf("get device at index %d: %s", i, nvml.ErrorString(ret))
		}

		processInfos, ret := device.GetProcessUtilization(0)
		if ret == nvml.ERROR_NOT_FOUND {
			continue
		}
		if ret != nvml.SUCCESS {
			return fmt.Errorf("get process utilization at index %d: %s", i, nvml.ErrorString(ret))
		}

		for _, processInfo := range processInfos {
			// Skip other GPU processes that don't consume a lot of resources
			if processInfo.SmUtil+processInfo.MemUtil+processInfo.DecUtil+processInfo.EncUtil < 2 {
				continue
			}
			if processInfo.Pid != p.pid {
				panic("other workload PIDs are using this device")
			}
			// Found our child PID running on the device, add to the used devices
			p.usedDevices[i] = struct{}{}
		}

		powerMilliWatts, ret := device.GetPowerUsage()
		if ret != nvml.SUCCESS {
			return fmt.Errorf("get power usage: %s", nvml.ErrorString(ret))
		}

		// Integrate energy
		power := float64(powerMilliWatts) * 1e-6 // kW
		dE := power * p.interval.Hours()
		p.totalEnergy += dE
	}
	return nil
}

func (p *EnergyPoller) Stop() error {
	p.ticker.Stop()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	ret := nvml.Shutdown()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("shutdown NVML: %s", nvml.ErrorString(ret))
	}
	return nil
}

func (p *EnergyPoller) DeviceName() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.deviceName
}

func (p *EnergyPoller) UsedDevices() []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	usedDevices := []string{}
	for k, _ := range p.usedDevices {
		usedDevices = append(usedDevices, strconv.Itoa(k))
	}
	return usedDevices
}

func (p *EnergyPoller) TotalEnergy() float64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.totalEnergy
}
