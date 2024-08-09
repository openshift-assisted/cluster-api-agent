package e2e

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-assisted/cluster-api-agent/test/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"libvirt.org/go/libvirt"
	"libvirt.org/libvirt-go-xml"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"
	"time"
)

const (
	networkName = "bmh"
	namespace   = "test-capi"
)

func SetupLibvirtNetwork(conn *libvirt.Connect) (*libvirt.Network, error) {
	network := getNetwork()
	networkXML, err := network.Marshal()
	if err != nil {
		return nil, err
	}
	return conn.NetworkCreateXML(networkXML)
}

func CleanupSushyTools(username, host string) error {
	dockerRunCmd := fmt.Sprintf(`podman rm -f sushy-tools`)
	cmd := exec.Command("ssh", "-A", fmt.Sprintf("%s@%s", username, host), dockerRunCmd)
	out, err := cmd.CombinedOutput()
	Expect(out).NotTo(BeEmpty())
	return err
}

func SetupSushyTools(username, host string) error {
	sushyConf := `
# Listen on 192.168.222.1:8000
SUSHY_EMULATOR_LISTEN_IP = u"192.168.222.1"
SUSHY_EMULATOR_LISTEN_PORT = 8000
# The libvirt URI to use. This option enables libvirt driver.
SUSHY_EMULATOR_LIBVIRT_URI = u"qemu:///system"
SUSHY_EMULATOR_IGNORE_BOOT_DEVICE = True
`
	sushyToolsConfPath := "/tmp/sushy-emulator.conf"
	dockerRunCmd := fmt.Sprintf(`podman run --name sushy-tools --rm --network host --privileged -d \
	  -v /var/run/libvirt:/var/run/libvirt:z \
	  -v "%s:/etc/sushy/sushy-emulator.conf:z" \
	  -e SUSHY_EMULATOR_CONFIG=/etc/sushy/sushy-emulator.conf \
	  quay.io/metal3-io/sushy-tools:latest sushy-emulator
`, sushyToolsConfPath)
	createConfigCmd := fmt.Sprintf("echo '%s' > %s; ", sushyConf, sushyToolsConfPath)

	cmd := exec.Command("ssh", "-A", fmt.Sprintf("%s@%s", username, host), createConfigCmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())

	cmd = exec.Command("ssh", "-A", fmt.Sprintf("%s@%s", username, host), dockerRunCmd)
	out, err := cmd.CombinedOutput()

	if err != nil {
		Expect(string(out)).To(BeEmpty())
		return err
	}
	Expect(err).NotTo(HaveOccurred())
	return cmd.Err
}

func InstallIronic() error {

	ironicManifestPath := "./test/e2e/manifests/ironic"
	cmd := exec.Command("kubectl", "apply", "-k", ironicManifestPath)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())
	return nil
}

func InstallBMO() error {
	bmoManifestPath := "./test/e2e/manifests/bmo"
	cmd := exec.Command("kubectl", "apply", "-k", bmoManifestPath)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())
	return nil
}

var _ = Describe("Libvirt Test", func() {
	var (
		ctx       context.Context = context.Background()
		username  string          = "root"
		hostname  string          = "rdu-infra-edge-03.infra-edge.lab.eng.rdu2.redhat.com"
		conn      *libvirt.Connect
		network   *libvirt.Network
		k8sClient client.Client
	)
	BeforeEach(func() {
		var err error
		uri := fmt.Sprintf("qemu+ssh://%s@%s/system", username, hostname)
		conn, err = libvirt.NewConnect(uri)
		Expect(err).ToNot(HaveOccurred())
		scheme := runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
		k8sClient, err = client.New(config.GetConfigOrDie(), client.Options{
			Scheme: scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		// cleanup
		_ = CleanNetworks(conn)
		_ = CleanupDomains(conn)

		// Setup Network
		network, err = SetupLibvirtNetwork(conn)
		Expect(err).NotTo(HaveOccurred())

		// spin up sushy tools
		Expect(SetupSushyTools(username, hostname)).To(Succeed())
	})
	AfterEach(func() {
		ns := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		Expect(k8sClient.Delete(ctx, &ns)).To(Succeed())

		// Delete sushy emulator
		_ = CleanupSushyTools(username, hostname)

		// destroy libvirt network
		Expect(network.Destroy()).To(Succeed())

		// destroy all libvirt domains
		_, err := conn.Close()
		Expect(err).NotTo(HaveOccurred())
	})
	It("libvirt go", func() {
		//kind create cluster --config kind.yaml --name openshift-capi

		Expect(utils.InstallCertManager()).To(Succeed())

		Expect(InstallIronic()).To(Succeed())
		Expect(InstallBMO()).To(Succeed())
		Expect(InstallInfrastructureOperator()).To(Succeed())

		Expect(InstallCoreCAPI()).To(Succeed())
		Expect(InstallCAPIProviders()).To(Succeed())

		/// Connect to K8s
		ns := v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(ctx, &ns)).To(Succeed())

		// get all MACs
		macs := []string{"00:60:2f:31:81:01", "00:60:2f:31:81:02", "00:60:2f:31:81:03", "00:60:2f:31:81:04", "00:60:2f:31:81:05"}
		nameToMac := map[string]string{}

		// Setup VMs
		for i, macAddress := range macs {
			domainName := fmt.Sprintf("bmh-vm-0%d", i+1)
			_, err := CreateDomain(conn, domainName, macAddress)
			nameToMac[domainName] = macAddress
			Expect(err).NotTo(HaveOccurred())
		}
		systemIDs, err := getRedfishIDs(username, hostname)
		Expect(err).NotTo(HaveOccurred())

		// Setup BMHs
		for _, systemID := range systemIDs {
			name, err := getSystemName(username, hostname, systemID)
			Expect(err).NotTo(HaveOccurred())
			if macAddress, ok := nameToMac[name]; ok {
				Expect(CreateBMH(ctx, k8sClient, namespace, name, macAddress, systemID)).To(Succeed())

			}
		}

		Expect(InstallExampleCluster()).To(Succeed())
		time.Sleep(300 * time.Second)
	})
})

func InstallExampleCluster() error {
	multiNodeExample := "./examples/multi-node-example.yaml"
	cmd := exec.Command("kubectl", "apply", "-f", multiNodeExample)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())
	return nil
}

func getSystemName(username, host, id string) (string, error) {
	getHostNameCmd := fmt.Sprintf(`curl -s 192.168.222.1:8000%s | jq -r '.Name'`, id)
	cmd := exec.Command("ssh", "-A", fmt.Sprintf("%s@%s", username, host), getHostNameCmd)
	out, err := cmd.CombinedOutput()
	name := strings.Trim(string(out), "\n")
	return name, err
}

func getRedfishIDs(username, host string) ([]string, error) {
	getRedfishIDsCmd := fmt.Sprintf(`curl -s 192.168.222.1:8000/redfish/v1/Systems | jq -r '.Members[]."@odata.id"'`)
	cmd := exec.Command("ssh", "-A", fmt.Sprintf("%s@%s", username, host), getRedfishIDsCmd)
	out, err := cmd.CombinedOutput()
	ids := strings.Split(strings.Trim(string(out), "\n"), "\n")
	return ids, err
}

func CreateBMH(ctx context.Context, client client.Client, namespace, name, macAddress, systemID string) error {
	bmh := v1alpha1.BareMetalHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BareMetalHost",
			APIVersion: "metal3.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"inspect.metal3.io": "disabled",
			},
		},
		Spec: v1alpha1.BareMetalHostSpec{
			BMC: v1alpha1.BMCDetails{
				Address:                        fmt.Sprintf("redfish-virtualmedia+http://192.168.222.1:8000%s", systemID),
				CredentialsName:                fmt.Sprintf("%s-secret", name),
				DisableCertificateVerification: false,
			},
			BootMode:       "UEFI",
			BootMACAddress: macAddress,
			Online:         true,
		},
	}
	return client.Create(ctx, &bmh)
}

func CreateDomain(conn *libvirt.Connect, domainName, macAddress string) (*libvirt.Domain, error) {
	domain := getDomain(domainName, macAddress)
	domainXML, err := domain.Marshal()
	if err != nil {
		return nil, err
	}
	d, err := conn.DomainCreateXML(domainXML, 0)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func InstallMetalLB() error {
	manifest := `apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: example
  namespace: metallb-system
spec:
  addresses:
  - 10.89.0.0-10.89.0.200
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: empty
  namespace: metallb-system`

	cmd := exec.Command("kubectl", "apply", "-f", "https://raw.githubusercontent.com/metallb/metallb/v0.13.7/config/manifests/metallb-native.yaml")
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())

	cmd = exec.Command("echo", manifest, "|", "kubectl", "apply", "-f", "-")
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())
	return nil
}

func InstallInfrastructureOperator() error {

	ironicManifestPath := "./test/e2e/manifests/infrastructure-operator"
	cmd := exec.Command("kubectl", "apply", "-k", ironicManifestPath)
	_, err := cmd.CombinedOutput()
	//Expect(string(out)).To(Equal(""))
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())
	return nil
}
func InstallAssistedService() error {
	if err := CloneAssistedServiceRepository(); err != nil {
		return err
	}
	cmd := exec.Command("cd", "assisted-service")
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return err
	}
	return nil
}

func CloneAssistedServiceRepository() error {
	cmd := exec.Command("git", "clone", "-f", "git@github.com:openshift/assisted-service.git")
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return err
	}
	cmd = exec.Command("cd", "assisted-service")
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return err
	}

	cmd = exec.Command("kubectl", "create", "namespace", "assisted-installer", "||", "true")
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return err
	}
	kustomization := `namespace: assisted-installer
resources:
	- ./ocp
	- leader_election_role.yaml
	- leader_election_role_binding.yaml
	- role.yaml
	- role_binding.yaml
`
	cmd = exec.Command("echo", kustomization, ">", "config/rbac/kustomization.yaml")
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return err
	}

	cmd = exec.Command("kustomize", "build", "config/rbac", "|", "kubectl", "apply", "-f", "-")
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return err
	}
	cmd = exec.Command("export", "ENABLE_KUBE_API=true")
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return err
	}
	cmd = exec.Command("make", "deploy-all")
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return err
	}
	return nil
}
func InstallAssistedInstaller() error {
	if err := InstallMetalLB(); err != nil {
		return err
	}
	if err := InstallAssistedService(); err != nil {
		return err
	}
	return nil
}

func InstallCAPIBootstrapProvider() error {
	cmd := exec.Command("kubectl", "apply", "-f", "bootstrap-components.yaml")
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())
	return nil
}

func InstallCAPIControlPlaneProvider() error {
	cmd := exec.Command("kubectl", "apply", "-f", "controlplane-components.yaml")
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())
	return nil
}

func InstallCAPIProviders() error {
	if err := InstallCAPIBootstrapProvider(); err != nil {
		return err
	}
	if err := InstallCAPIControlPlaneProvider(); err != nil {
		return err
	}
	return nil
}

func InstallCoreCAPI() error {
	cmd := exec.Command("clusterctl", "init", "--core", "cluster-api:v1.7.1", "--bootstrap", "-", "--control-plane", "-", "--infrastructure", "metal3:v1.6.0")
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return err
	}
	Expect(err).NotTo(HaveOccurred())
	return nil
}

func CleanupDomains(conn *libvirt.Connect) error {
	domains, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return err
	}
	for _, domain := range domains {
		domainName, err := domain.GetName()
		if err != nil {
			continue
		}
		if strings.HasPrefix(domainName, "bmh-vm") {
			domain.Destroy()
			domain.Undefine()
		}
	}
	return nil
}

func getDomain(domainName string, macAddress string) libvirtxml.Domain {
	return libvirtxml.Domain{
		Type:     "kvm",
		Name:     domainName,
		Metadata: nil,
		Memory: &libvirtxml.DomainMemory{
			Value: 16,
			Unit:  "GB",
		},
		VCPU: &libvirtxml.DomainVCPU{Placement: "static", Value: 8},
		//VCPUs:                nil,
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Arch:    "x86_64", //
				Machine: "pc-q35-rhel8.6.0",
				Type:    "hvm",
			},
			Firmware:     "",
			FirmwareInfo: nil,
			Init:         "",
			InitArgs:     nil,
			InitEnv:      nil,
			InitDir:      "",
			InitUser:     "",
			InitGroup:    "",
			Loader:       nil,
			NVRam:        nil,
			Kernel:       "",
			Initrd:       "",
			Cmdline:      "",
			DTB:          "",
			ACPI:         nil,
			BootDevices:  nil,
			BootMenu:     nil,
			BIOS:         nil,
			SMBios:       nil,
		},
		//Features:             nil,
		//CPU: nil,
		/*Devices: &libvirtxml.DomainDeviceList{
			Emulator: "",
			Disks: []libvirtxml.DomainDisk{
				{
					XMLName:       xml.Name{},
					Device:        "",
					RawIO:         "",
					SGIO:          "",
					Snapshot:      "",
					Model:         "",
					Driver:        nil,
					Auth:          nil,
					Source:        nil,
					BackingStore:  nil,
					BackendDomain: nil,
					Geometry:      nil,
					BlockIO:       nil,
					Mirror:        nil,
					Target:        nil,
					IOTune:        nil,
					ReadOnly:      nil,
					Shareable:     nil,
					Transient:     nil,
					Serial:        "",
					WWN:           "",
					Vendor:        "",
					Product:       "",
					Encryption:    nil,
					Boot:          nil,
					ACPI:          nil,
					Alias:         nil,
					Address:       nil,
				},
			},
			Controllers:  nil,
			Leases:       nil,
			Filesystems:  nil,
			Interfaces:   nil,
			Smartcards:   nil,
			Serials:      nil,
			Parallels:    nil,
			Consoles:     nil,
			Channels:     nil,
			Inputs:       nil,
			TPMs:         nil,
			Graphics:     nil,
			Sounds:       nil,
			Audios:       nil,
			Videos:       nil,
			Hostdevs:     nil,
			RedirDevs:    nil,
			RedirFilters: nil,
			Hubs:         nil,
			Watchdog:     nil,
			MemBalloon:   nil,
			RNGs:         nil,
			NVRAM:        nil,
			Panics:       nil,
			Shmems:       nil,
			Memorydevs:   nil,
			IOMMU:        nil,
			VSock:        nil,
		},*/
		SecLabel:             nil,
		KeyWrap:              nil,
		LaunchSecurity:       nil,
		QEMUCommandline:      nil,
		QEMUCapabilities:     nil,
		QEMUDeprecation:      nil,
		LXCNamespace:         nil,
		BHyveCommandline:     nil,
		VMWareDataCenterPath: nil,
		XenCommandline:       nil,
	}
}

func CleanNetworks(conn *libvirt.Connect) error {
	nets, err := conn.ListAllNetworks(libvirt.CONNECT_LIST_NETWORKS_ACTIVE | libvirt.CONNECT_LIST_NETWORKS_INACTIVE)
	if err != nil {
		return err
	}
	for _, net := range nets {
		name, err := net.GetName()
		if err != nil {
			return err
		}
		if name == networkName {
			Expect(net.Destroy()).To(Succeed())
			return nil
		}
	}
	return nil
}

func getNetwork() libvirtxml.Network {
	return libvirtxml.Network{
		Name: networkName,
		Forward: &libvirtxml.NetworkForward{
			Mode: "nat",
		},
		Bridge: &libvirtxml.NetworkBridge{
			Name:  networkName,
			STP:   "on",
			Delay: "0",
		},
		MAC: &libvirtxml.NetworkMAC{Address: "52:54:00:fb:8a:5f"},
		IPs: []libvirtxml.NetworkIP{
			{
				Address: "192.168.222.1",
				Netmask: "255.255.255.0",
				DHCP: &libvirtxml.NetworkDHCP{
					Ranges: []libvirtxml.NetworkDHCPRange{
						{
							Start: "192.168.222.2",
							End:   "192.168.222.254",
						},
					},
					Hosts: []libvirtxml.NetworkDHCPHost{
						{
							MAC:  "00:60:2f:31:81:01",
							Name: "okd-1",
							IP:   "192.168.222.30",
						},
						{
							MAC:  "00:60:2f:31:81:02",
							Name: "okd-2",
							IP:   "192.168.222.31",
						},
						{
							MAC:  "00:60:2f:31:81:03",
							Name: "okd-3",
							IP:   "192.168.222.32",
						},
						{
							MAC:  "00:60:2f:31:81:04",
							Name: "okd-4",
							IP:   "192.168.222.33",
						},
						{
							MAC:  "00:60:2f:31:81:05",
							Name: "okd-5",
							IP:   "192.168.222.34",
						},
					},
				},
			},
		},
		DnsmasqOptions: &libvirtxml.NetworkDnsmasqOptions{
			XMLName: xml.Name{},
			Option: []libvirtxml.NetworkDnsmasqOption{
				{Value: "local=/lab.home/"},
				{Value: "address=/api.test-sno.lab.home/192.168.222.31"},
				{Value: "address=/api-int.test-sno.lab.home/192.168.222.31"},
				{Value: "address=/.apps.test-sno.lab.home/192.168.222.31"},
				{Value: "address=/api.test-multinode.lab.home/192.168.222.40"},
				{Value: "address=/api-int.test-multinode.lab.home/192.168.222.40"},
				{Value: "address=/.apps.test-multinode.lab.home/192.168.222.41"},
			},
		},
	}
}
