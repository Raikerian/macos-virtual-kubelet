package provider

import (
	"context"
	"fmt"
	"io"

	"net"

	dto "github.com/prometheus/client_model/go"
	"github.com/raikerian/macos-virtual-kubelet/internal/manager"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/node/api/statsv1alpha1"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

// MacOSProvider implements the virtual-kubelet provider interface.
type MacOSProvider struct {
	nodeutil.Provider
	rm                 *manager.ResourceManager
	nodeName           string
	operatingSystem    string
	internalIP         string
	daemonEndpointPort int32
}

const (
	DefaultPods = 110
)

var (
	errNotImplemented = fmt.Errorf("not implemented by MacOS provider")
)

// NewMacOSProvider creates a new MacOS provider.
func NewMacOSProvider(rm *manager.ResourceManager, nodeName, operatingSystem, internalIP string, daemonEndpointPort int32) *MacOSProvider {
	return &MacOSProvider{
		rm:                 rm,
		nodeName:           nodeName,
		operatingSystem:    operatingSystem,
		internalIP:         internalIP,
		daemonEndpointPort: daemonEndpointPort,
	}
}

// CreatePod takes a Kubernetes Pod and deploys it within the MacOS provider.
func (p *MacOSProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	log.G(ctx).Infof("Received CreatePod request for %s/%s.\n", pod.Namespace, pod.Name)
	return p.rm.AddPod(ctx, pod)
}

// UpdatePod takes a Kubernetes Pod and updates it within the provider.
func (p *MacOSProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	log.G(ctx).Infof("Received UpdatePod request for %s/%s.\n", pod.Namespace, pod.Name)

	return errNotImplemented
}

// DeletePod takes a Kubernetes Pod and deletes it from the provider.
func (p *MacOSProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	log.G(ctx).Infof("Received DeletePod request for %s/%s.\n", pod.Namespace, pod.Name)
	return p.rm.DeletePod(ctx, pod)
}

// GetPod retrieves a pod by name from the provider (can be cached).
func (p *MacOSProvider) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	log.G(ctx).Infof("Received GetPod request for %s/%s.\n", namespace, name)
	return p.rm.GetPod(types.NamespacedName{Namespace: namespace, Name: name}), nil
}

// GetPodStatus retrieves the status of a pod by name from the provider.
func (p *MacOSProvider) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	log.G(ctx).Infof("Received GetPodStatus request for %s/%s.\n", namespace, name)
	return nil, errNotImplemented
	// pod, err := p.GetPod(ctx, namespace, name)
	// if err != nil {
	// 	return nil, err
	// }

	// if pod == nil {
	// 	return nil, nil
	// }

	// return &pod.Status, nil
}

// GetPods retrieves a list of all pods running on the provider (can be cached).
func (p *MacOSProvider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	log.G(ctx).Info("Received GetPods request.\n")
	return p.rm.GetPods(), nil
}

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *MacOSProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	log.G(ctx).Infof("Received GetContainerLogs request for %s/%s/%s.\n", namespace, podName, containerName)
	return nil, errNotImplemented
}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *MacOSProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error {
	log.G(ctx).Infof("Received RunInContainer request for %s/%s/%s.\n", namespace, podName, containerName)
	return errNotImplemented
}

// AttachToContainer attaches to the executing process of a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *MacOSProvider) AttachToContainer(ctx context.Context, namespace, podName, containerName string, attach api.AttachIO) error {
	log.G(ctx).Infof("Received AttachToContainer request for %s/%s/%s.\n", namespace, podName, containerName)
	return errNotImplemented
}

// GetStatsSummary gets the stats for the node, including running pods
func (p *MacOSProvider) GetStatsSummary(ctx context.Context) (*statsv1alpha1.Summary, error) {
	log.G(ctx).Info("Received GetStatsSummary request.\n")
	return nil, errNotImplemented
}

// GetMetricsResource gets the metrics for the node, including running pods
func (p *MacOSProvider) GetMetricsResource(ctx context.Context) ([]*dto.MetricFamily, error) {
	log.G(ctx).Info("Received GetMetricsResource request.\n")
	return nil, errNotImplemented
}

// PortForward forwards a local port to a port on the pod
func (p *MacOSProvider) PortForward(ctx context.Context, namespace, pod string, port int32, stream io.ReadWriteCloser) error {
	log.G(ctx).Infof("Received PortForward request for %s/%s:%d.\n", namespace, pod, port)
	return errNotImplemented
}

func (p *MacOSProvider) ConfigureNode(ctx context.Context, n *corev1.Node) {
	// if p.config.ProviderID != "" {
	// 	n.Spec.ProviderID = p.config.ProviderID
	// }
	capacity := p.capacity(ctx)
	n.Status.Capacity = capacity
	n.Status.Allocatable = capacity
	// n.Status.Conditions = p.nodeConditions()
	n.Status.Addresses = p.nodeAddresses(ctx)
	n.Status.DaemonEndpoints = corev1.NodeDaemonEndpoints{
		KubeletEndpoint: corev1.DaemonEndpoint{
			Port: p.daemonEndpointPort,
		},
	}
	n.Status.NodeInfo = p.nodeInfo(ctx)
	// n.ObjectMeta.Labels["alpha.service-controller.kubernetes.io/exclude-balancer"] = "true"
	// n.ObjectMeta.Labels["node.kubernetes.io/exclude-from-external-load-balancers"] = "true"
}

// Capacity returns a resource list containing the capacity limits.
func (p *MacOSProvider) capacity(ctx context.Context) corev1.ResourceList {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		log.G(ctx).WithError(err).Error("Error getting memory capacity")
	}

	c, err := cpu.CountsWithContext(ctx, true)
	if err != nil {
		log.G(ctx).WithError(err).Error("Error getting cpu capacity")
	}

	d, err := disk.UsageWithContext(ctx, "/")
	if err != nil {
		log.G(ctx).WithError(err).Error("Error getting disk capacity")
	}

	rl := corev1.ResourceList{
		"cpu":               *resource.NewQuantity(int64(c), resource.DecimalSI),
		"memory":            *resource.NewQuantity(int64(v.Total), resource.BinarySI),
		"ephemeral-storage": *resource.NewQuantity(int64(d.Total), resource.BinarySI),
		"pods":              *resource.NewQuantity(DefaultPods, resource.DecimalSI),
	}
	return rl
}

func (p *MacOSProvider) nodeAddresses(ctx context.Context) []corev1.NodeAddress {
	ifs, err := psnet.InterfacesWithContext(ctx)
	if err != nil {
		log.G(ctx).WithError(err).Error("Error getting network interfaces")
	}

	addr := ""
	for _, i := range ifs {
		// en0 is a default interface on Apple Silicon machines
		// for now, assuming that all machines provided are act as so
		if i.Name == "en0" {
			for _, a := range i.Addrs {
				ip, _, err := net.ParseCIDR(a.Addr)
				if err != nil {
					log.G(ctx).WithError(err).Error("Error parsing CIDR")
				}
				if ip.To4() != nil {
					addr = ip.String()
				}
			}
			break
		}
	}

	return []corev1.NodeAddress{
		{
			Type:    corev1.NodeInternalIP,
			Address: addr,
		},
		{
			Type:    corev1.NodeHostName,
			Address: p.nodeName,
		},
	}
}

func (p *MacOSProvider) nodeInfo(ctx context.Context) corev1.NodeSystemInfo {
	info, err := host.InfoWithContext(ctx)
	if err != nil {
		log.G(ctx).WithError(err).Error("Error getting host info")
	}

	return corev1.NodeSystemInfo{
		MachineID:               info.HostID,
		KernelVersion:           info.KernelVersion,
		OSImage:                 info.OS,
		ContainerRuntimeVersion: "",
		OperatingSystem:         p.operatingSystem + " " + info.PlatformVersion,
		Architecture:            info.KernelArch,
	}
}
