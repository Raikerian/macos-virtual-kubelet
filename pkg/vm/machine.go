package vm

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/virtual-kubelet/virtual-kubelet/log"
)

type Manager struct {
	vm *vz.VirtualMachine
}

func NewMacMachine(cpuCount uint, memorySize uint64) (*Manager, error) {
	platformConfig, err := SetupMacPlatformConfiguration()
	if err != nil {
		return nil, err
	}

	if cpuCount == 0 {
		// cpu count wasnt provided, compute the basic one
		cpuCount = ComputeCPUCount()
	}
	if memorySize == 0 {
		// memory size wasnt provided, compute the basic one
		memorySize = ComputeMemorySize()
	}

	// bridge physical interface en0
	// en0 is the default interface on Apple Silicon Macs
	config, err := CreateVMConfiguration(platformConfig, cpuCount, memorySize, "en0")
	if err != nil {
		return nil, err
	}

	vm, err := vz.NewVirtualMachine(config)
	if err != nil {
		return nil, err
	}

	return &Manager{
		vm: vm,
	}, nil
}

func (m *Manager) Run(ctx context.Context, isGUI bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := m.vm.Start(); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		for {
			select {
			case newState := <-m.vm.StateChangedNotify():
				if newState == vz.VirtualMachineStateRunning {
					log.G(ctx).Info("VM is running")
					errCh <- nil
					return
				}
			case err := <-errCh:
				errCh <- fmt.Errorf("failed to start vm: %w", err)
				return
			}
		}
	}()

	if isGUI {
		// start GUI
		runtime.LockOSThread()
		m.vm.StartGraphicApplication(960, 600)
		runtime.UnlockOSThread()
	}

	return <-errCh
}

func (m *Manager) Stop(ctx context.Context) error {
	for i := 1; m.vm.CanRequestStop(); i++ {
		result, err := m.vm.RequestStop()
		log.G(ctx).Infof("sent stop request(%d): %t, %v", i, result, err)
		time.Sleep(time.Second * 3)
		if i > 3 {
			log.G(ctx).Info("VM cannot be stopped at this moment")
			if err := m.vm.Stop(); err != nil {
				log.G(ctx).WithError(err).Error("Error stopping VM")
			}
		}
	}
	return nil
}
