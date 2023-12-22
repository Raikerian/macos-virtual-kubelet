package vm

import (
	"fmt"
	"os"

	"github.com/Code-Hex/vz/v3"
)

func SetupMacPlatformConfiguration() (*vz.MacPlatformConfiguration, error) {
	auxiliaryStorage, err := vz.NewMacAuxiliaryStorage(GetAuxiliaryStoragePath())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new mac auxiliary storage: %w", err)
	}
	hardwareModel, err := vz.NewMacHardwareModelWithDataPath(
		GetHardwareModelPath(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new hardware model: %w", err)
	}
	machineIdentifier, err := vz.NewMacMachineIdentifierWithDataPath(
		GetMachineIdentifierPath(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new machine identifier: %w", err)
	}
	return vz.NewMacPlatformConfiguration(
		vz.WithMacAuxiliaryStorage(auxiliaryStorage),
		vz.WithMacHardwareModel(hardwareModel),
		vz.WithMacMachineIdentifier(machineIdentifier),
	)
}

func CreateVMConfiguration(platformConfig vz.PlatformConfiguration, cpuCount uint, memorySize uint64, networkInterfaceIdentifier string) (*vz.VirtualMachineConfiguration, error) {
	// verify cpu count
	if cpuCount > vz.VirtualMachineConfigurationMaximumAllowedCPUCount() {
		return nil, fmt.Errorf("cpu count is too large: %d", cpuCount)
	}
	if cpuCount < vz.VirtualMachineConfigurationMinimumAllowedCPUCount() {
		return nil, fmt.Errorf("cpu count is too small: %d", cpuCount)
	}

	// verify memory size
	if memorySize > vz.VirtualMachineConfigurationMaximumAllowedMemorySize() {
		return nil, fmt.Errorf("memory size is too large: %d", memorySize)
	}
	if memorySize < vz.VirtualMachineConfigurationMinimumAllowedMemorySize() {
		return nil, fmt.Errorf("memory size is too small: %d", memorySize)
	}

	bootloader, err := vz.NewMacOSBootLoader()
	if err != nil {
		return nil, err
	}

	config, err := vz.NewVirtualMachineConfiguration(
		bootloader,
		cpuCount,
		memorySize,
	)
	if err != nil {
		return nil, err
	}
	config.SetPlatformVirtualMachineConfiguration(platformConfig)
	graphicsDeviceConfig, err := CreateGraphicsDeviceConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create graphics device configuration: %w", err)
	}
	config.SetGraphicsDevicesVirtualMachineConfiguration([]vz.GraphicsDeviceConfiguration{
		graphicsDeviceConfig,
	})
	blockDeviceConfig, err := CreateBlockDeviceConfiguration(GetDiskImagePath())
	if err != nil {
		return nil, fmt.Errorf("failed to create block device configuration: %w", err)
	}
	config.SetStorageDevicesVirtualMachineConfiguration([]vz.StorageDeviceConfiguration{blockDeviceConfig})

	var networkInterface vz.BridgedNetwork
	if networkInterfaceIdentifier != "" {
		networkInterfaces := vz.NetworkInterfaces()
		for _, b := range networkInterfaces {
			if b.Identifier() == networkInterfaceIdentifier {
				networkInterface = b
				break
			}
		}
		if networkInterface == nil {
			return nil, fmt.Errorf("network interface %s not found", networkInterfaceIdentifier)
		}
	}
	networkDeviceConfig, err := CreateNetworkDeviceConfiguration(networkInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to create network device configuration: %w", err)
	}
	config.SetNetworkDevicesVirtualMachineConfiguration([]*vz.VirtioNetworkDeviceConfiguration{
		networkDeviceConfig,
	})

	usbScreenPointingDevice, err := vz.NewUSBScreenCoordinatePointingDeviceConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create pointing device configuration: %w", err)
	}
	pointingDevices := []vz.PointingDeviceConfiguration{usbScreenPointingDevice}

	trackpad, err := vz.NewMacTrackpadConfiguration()
	if err == nil {
		pointingDevices = append(pointingDevices, trackpad)
	}
	config.SetPointingDevicesVirtualMachineConfiguration(pointingDevices)

	keyboardDeviceConfig, err := CreateKeyboardConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create keyboard device configuration: %w", err)
	}
	config.SetKeyboardsVirtualMachineConfiguration([]vz.KeyboardConfiguration{
		keyboardDeviceConfig,
	})

	audioDeviceConfig, err := CreateAudioDeviceConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create audio device configuration: %w", err)
	}
	config.SetAudioDevicesVirtualMachineConfiguration([]vz.AudioDeviceConfiguration{
		audioDeviceConfig,
	})

	validated, err := config.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate configuration: %w", err)
	}
	if !validated {
		return nil, fmt.Errorf("invalid configuration")
	}

	return config, nil
}

func CreateGraphicsDeviceConfiguration() (*vz.MacGraphicsDeviceConfiguration, error) {
	graphicDeviceConfig, err := vz.NewMacGraphicsDeviceConfiguration()
	if err != nil {
		return nil, err
	}
	graphicsDisplayConfig, err := vz.NewMacGraphicsDisplayConfiguration(1920, 1200, 80)
	if err != nil {
		return nil, err
	}
	graphicDeviceConfig.SetDisplays(
		graphicsDisplayConfig,
	)
	return graphicDeviceConfig, nil
}

func CreateBlockDeviceConfiguration(diskPath string) (*vz.VirtioBlockDeviceConfiguration, error) {
	// create disk image with 128 GiB
	if err := vz.CreateDiskImage(diskPath, 128*1024*1024*1024); err != nil {
		if !os.IsExist(err) {
			return nil, fmt.Errorf("failed to create disk image: %w", err)
		}
	}

	diskImageAttachment, err := vz.NewDiskImageStorageDeviceAttachment(
		diskPath,
		false,
	)
	if err != nil {
		return nil, err
	}
	return vz.NewVirtioBlockDeviceConfiguration(diskImageAttachment)
}

func CreateNetworkDeviceConfiguration(networkInterface vz.BridgedNetwork) (*vz.VirtioNetworkDeviceConfiguration, error) {
	var attachment vz.NetworkDeviceAttachment
	var err error
	if networkInterface != nil {
		attachment, err = vz.NewBridgedNetworkDeviceAttachment(networkInterface)
		if err != nil {
			return nil, err
		}
	} else {
		attachment, err = vz.NewNATNetworkDeviceAttachment()
		if err != nil {
			return nil, err
		}
	}

	return vz.NewVirtioNetworkDeviceConfiguration(attachment)
}

func CreateKeyboardConfiguration() (*vz.USBKeyboardConfiguration, error) {
	return vz.NewUSBKeyboardConfiguration()
}

func CreateAudioDeviceConfiguration() (*vz.VirtioSoundDeviceConfiguration, error) {
	audioConfig, err := vz.NewVirtioSoundDeviceConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create sound device configuration: %w", err)
	}
	inputStream, err := vz.NewVirtioSoundDeviceHostInputStreamConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create input stream configuration: %w", err)
	}
	outputStream, err := vz.NewVirtioSoundDeviceHostOutputStreamConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create output stream configuration: %w", err)
	}
	audioConfig.SetStreams(
		inputStream,
		outputStream,
	)
	return audioConfig, nil
}
