package manager

import (
	"context"

	"golang.org/x/exp/maps"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1listers "k8s.io/client-go/listers/core/v1"

	"github.com/Code-Hex/vz/v3"
	"github.com/raikerian/macos-virtual-kubelet/pkg/vm"
	"github.com/virtual-kubelet/virtual-kubelet/log"
)

// ResourceManager acts as a passthrough to a cache (lister) for pods assigned to the current node.
// It is also a passthrough to a cache (lister) for Kubernetes secrets and config maps.
type ResourceManager struct {
	pods      map[types.NamespacedName]*v1.Pod
	instances map[types.UID]*vz.VirtualMachine

	// potentially not needed listers
	podLister       corev1listers.PodLister
	secretLister    corev1listers.SecretLister
	configMapLister corev1listers.ConfigMapLister
	serviceLister   corev1listers.ServiceLister
}

// NewResourceManager returns a ResourceManager with the internal maps initialized.
func NewResourceManager(podLister corev1listers.PodLister, secretLister corev1listers.SecretLister, configMapLister corev1listers.ConfigMapLister, serviceLister corev1listers.ServiceLister) (*ResourceManager, error) {
	rm := ResourceManager{
		pods:      map[types.NamespacedName]*v1.Pod{},
		instances: map[types.UID]*vz.VirtualMachine{},

		podLister:       podLister,
		secretLister:    secretLister,
		configMapLister: configMapLister,
		serviceLister:   serviceLister,
	}
	return &rm, nil
}

func (rm *ResourceManager) CreatePod(ctx context.Context, pod *v1.Pod) error {
	uid := pod.GetUID()

	cpuSpec := pod.Spec.Containers[0].Resources.Requests[v1.ResourceCPU]
	memorySpec := pod.Spec.Containers[0].Resources.Requests[v1.ResourceMemory]

	cpu, ok := cpuSpec.AsInt64()
	if !ok {
		log.G(ctx).Warn("Failed to get CPU request")
	}
	memory, ok := memorySpec.AsInt64()
	if !ok {
		log.G(ctx).Warn("Failed to get memory request")
	}

	vm, err := createVirtualMachine(uint(cpu), uint64(memory))
	if err != nil {
		return err
	}

	if err := vm.Start(); err != nil {
		return err
	}

	rm.pods[types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}] = pod
	rm.instances[uid] = vm

	return nil
}

func (rm *ResourceManager) DeletePod(ctx context.Context, pod *v1.Pod) error {
	nm := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
	if rm.pods[nm] == nil {
		return nil
	}

	uid := pod.GetUID()
	vm := rm.instances[uid]

	if err := vm.Stop(); err != nil {
		return err
	}

	rm.pods[nm] = nil
	rm.instances[uid] = nil

	return nil
}

func (rm *ResourceManager) GetPod(nm types.NamespacedName) *v1.Pod {
	return rm.pods[nm]
}

func (rm *ResourceManager) GetPodStatus(nm types.NamespacedName) *v1.PodStatus {
	pod := rm.GetPod(nm)
	if pod == nil {
		return nil
	}

	vm := rm.instances[pod.GetUID()]
	switch vm.State() {
	case vz.VirtualMachineStateStarting:
		pod.Status.Phase = v1.PodPending
	case vz.VirtualMachineStateRunning, vz.VirtualMachineStateStopping:
		pod.Status.Phase = v1.PodRunning
		started := true
		pod.Status.ContainerStatuses = []v1.ContainerStatus{
			{
				Name:    pod.Spec.Containers[0].Name,
				State:   v1.ContainerState{Running: &v1.ContainerStateRunning{StartedAt: pod.CreationTimestamp}},
				Ready:   true,
				Started: &started,
			},
		}
	case vz.VirtualMachineStateStopped:
		pod.Status.Phase = v1.PodSucceeded
	case vz.VirtualMachineStateError:
		pod.Status.Phase = v1.PodFailed
	}

	return &pod.Status
}

func (rm *ResourceManager) GetPods() []*v1.Pod {
	return maps.Values(rm.pods)
}

// GetConfigMap retrieves the specified config map from the cache.
// func (rm *ResourceManager) GetConfigMap(name, namespace string) (*v1.ConfigMap, error) {
// 	return rm.configMapLister.ConfigMaps(namespace).Get(name)
// }

// // GetSecret retrieves the specified secret from Kubernetes.
// func (rm *ResourceManager) GetSecret(name, namespace string) (*v1.Secret, error) {
// 	return rm.secretLister.Secrets(namespace).Get(name)
// }

// // ListServices retrieves the list of services from Kubernetes.
// func (rm *ResourceManager) ListServices() ([]*v1.Service, error) {
// 	return rm.serviceLister.List(labels.Everything())
// }

func createVirtualMachine(cpuCount uint, memorySize uint64) (*vz.VirtualMachine, error) {
	platformConfig, err := vm.SetupMacPlatformConfiguration()
	if err != nil {
		return nil, err
	}

	if cpuCount == 0 {
		// cpu count wasnt provided, compute the basic one
		cpuCount = vm.ComputeCPUCount()
	}
	if memorySize == 0 {
		// memory size wasnt provided, compute the basic one
		memorySize = vm.ComputeMemorySize()
	}

	// bridge physical interface en0
	// en0 is the default interface on Apple Silicon Macs
	config, err := vm.CreateVMConfiguration(platformConfig, cpuCount, memorySize, "en0")
	if err != nil {
		return nil, err
	}

	vm, err := vz.NewVirtualMachine(config)
	if err != nil {
		return nil, err
	}

	return vm, nil
}
