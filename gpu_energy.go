package calcium

type GPUEnergyPoller interface {
	DeviceName() string
	UsedDevices() []string
	TotalEnergy() float64
	Stop() error
}
