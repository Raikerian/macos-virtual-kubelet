package manager

import (
	"context"

	"golang.org/x/exp/maps"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1listers "k8s.io/client-go/listers/core/v1"

	"github.com/raikerian/macos-virtual-kubelet/pkg/vm"
	"github.com/virtual-kubelet/virtual-kubelet/log"
)

// ResourceManager acts as a passthrough to a cache (lister) for pods assigned to the current node.
// It is also a passthrough to a cache (lister) for Kubernetes secrets and config maps.
type ResourceManager struct {
	pods      map[types.NamespacedName]*v1.Pod
	instances map[types.UID]*vm.Manager

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
		instances: map[types.UID]*vm.Manager{},

		podLister:       podLister,
		secretLister:    secretLister,
		configMapLister: configMapLister,
		serviceLister:   serviceLister,
	}
	return &rm, nil
}

func (rm *ResourceManager) GetPods() []*v1.Pod {
	return maps.Values(rm.pods)
}

func (rm *ResourceManager) GetPod(nm types.NamespacedName) *v1.Pod {
	return rm.pods[nm]
}

func (rm *ResourceManager) AddPod(ctx context.Context, pod *v1.Pod) error {
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

	m, err := vm.NewMacMachine(uint(cpu), uint64(memory))
	if err != nil {
		return err
	}

	err = m.Run(ctx, false)
	if err != nil {
		return err
	}

	rm.pods[types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}] = pod
	rm.instances[uid] = m

	return nil
}

func (rm *ResourceManager) DeletePod(ctx context.Context, pod *v1.Pod) error {
	if rm.pods[types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}] == nil {
		return nil
	}

	uid := pod.GetUID()
	m := rm.instances[uid]

	err := m.Stop(ctx)
	if err != nil {
		return err
	}

	rm.pods[types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}] = nil
	rm.instances[uid] = nil

	return nil
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
