package main

import (
	"fmt"
	"log"
	"bytes"
	"os"
	"strings"
	"net"
	"encoding/xml"

	"math/rand"

	libvirt "github.com/libvirt/libvirt-go"
	"github.com/libvirt/libvirt-go-xml"
	"github.com/davecgh/go-spew/spew"
)

// LibVirtConIsNil is a global string error msg
const (
	LibVirtConIsNil string = "the libvirt connection was nil"
	domainName             = "test"
	domainMemory           = 2048
	domainVcpu             = 1
	ignKey                 = "/var/lib/libvirt/images/master-bootstrap.ign"
	volumeKey 			   = "/var/lib/libvirt/images/extra-worker"
	networkInterfaceName = "tectonic"
	networkInterfaceHostname = "extra-worker"
	networkInterfaceAddress = "192.168.124.0/24"
	networkUUID = "11ac9efd-2f8b-455d-93a7-2027f35a83be"
	autostart = true
	uri = "qemu://10.80.94.1/system"
)

// Client libvirt
type Client struct {
	libvirt *libvirt.Connect
}

type pendingMapping struct {
	mac      string
	hostname string
	network  *libvirt.Network
}

func newDomainDef() libvirtxml.Domain {
	domainDef := libvirtxml.Domain{
		OS: &libvirtxml.DomainOS{
			Type: &libvirtxml.DomainOSType{
				Type: "hvm",
			},
		},
		Memory: &libvirtxml.DomainMemory{
			Unit:  "MiB",
			Value: 512,
		},
		VCPU: &libvirtxml.DomainVCPU{
			Placement: "static",
			Value:     1,
		},
		CPU: &libvirtxml.DomainCPU{},
		Devices: &libvirtxml.DomainDeviceList{
			Graphics: []libvirtxml.DomainGraphic{
				{
					Spice: &libvirtxml.DomainGraphicSpice{
						AutoPort: "yes",
					},
				},
			},
			Channels: []libvirtxml.DomainChannel{
				{
					Target: &libvirtxml.DomainChannelTarget{
						VirtIO: &libvirtxml.DomainChannelTargetVirtIO{
							Name: "org.qemu.guest_agent.0",
						},
					},
				},
			},
			RNGs: []libvirtxml.DomainRNG{
				{
					Model: "virtio",
					Backend: &libvirtxml.DomainRNGBackend{
						Random: &libvirtxml.DomainRNGBackendRandom{},
					},
				},
			},
		},
		Features: &libvirtxml.DomainFeatureList{
			PAE:  &libvirtxml.DomainFeature{},
			ACPI: &libvirtxml.DomainFeature{},
			APIC: &libvirtxml.DomainFeatureAPIC{},
		},
	}

	if v := os.Getenv("TERRAFORM_LIBVIRT_TEST_DOMAIN_TYPE"); v != "" {
		domainDef.Type = v
	} else {
		domainDef.Type = "kvm"
	}

	return domainDef
}

func getHostArchitecture(virConn *libvirt.Connect) (string, error) {
	type HostCapabilities struct {
		XMLName xml.Name `xml:"capabilities"`
		Host    struct {
			XMLName xml.Name `xml:"host"`
			CPU     struct {
				XMLName xml.Name `xml:"cpu"`
				Arch    string   `xml:"arch"`
			}
		}
	}

	info, err := virConn.GetCapabilities()
	if err != nil {
		return "", err
	}

	capabilities := HostCapabilities{}
	xml.Unmarshal([]byte(info), &capabilities)

	return capabilities.Host.CPU.Arch, nil
}

func getHostCapabilities(virConn *libvirt.Connect) (libvirtxml.Caps, error) {
	// We should perhaps think of storing this on the connect object
	// on first call to avoid the back and forth
	caps := libvirtxml.Caps{}
	capsXML, err := virConn.GetCapabilities()
	if err != nil {
		return caps, err
	}
	xml.Unmarshal([]byte(capsXML), &caps)
	log.Printf("[TRACE] Capabilities of host \n %+v", caps)
	return caps, nil
}

func getGuestForArchType(caps libvirtxml.Caps, arch string, virttype string) (libvirtxml.CapsGuest, error) {
	for _, guest := range caps.Guests {
		log.Printf("[TRACE] Checking for %s/%s against %s/%s\n", arch, virttype, guest.Arch.Name, guest.OSType)
		if guest.Arch.Name == arch && guest.OSType == virttype {
			log.Printf("[DEBUG] Found %d machines in guest for %s/%s", len(guest.Arch.Machines), arch, virttype)
			return guest, nil
		}
	}
	return libvirtxml.CapsGuest{}, fmt.Errorf("[DEBUG] Could not find any guests for architecure type %s/%s", virttype, arch)
}

func getCanonicalMachineName(caps libvirtxml.Caps, arch string, virttype string, targetmachine string) (string, error) {
	log.Printf("[INFO] getCanonicalMachineName")
	guest, err := getGuestForArchType(caps, arch, virttype)
	if err != nil {
		return "", err
	}

	for _, machine := range guest.Arch.Machines {
		if machine.Name == targetmachine {
			if machine.Canonical != "" {
				return machine.Canonical, nil
			}
			return machine.Name, nil
		}
	}
	return "", fmt.Errorf("[WARN] Cannot find machine type %s for %s/%s in %v", targetmachine, virttype, arch, caps)
}

func newDomainDefForConnection(virConn *libvirt.Connect) (libvirtxml.Domain, error) {
	d := newDomainDef()

	arch, err := getHostArchitecture(virConn)
	if err != nil {
		return d, err
	}
	d.OS.Type.Arch = arch

	caps, err := getHostCapabilities(virConn)
	if err != nil {
		return d, err
	}
	guest, err := getGuestForArchType(caps, d.OS.Type.Arch, d.OS.Type.Type)
	if err != nil {
		return d, err
	}

	d.Devices.Emulator = guest.Arch.Emulator

	if len(guest.Arch.Machines) > 0 {
		d.OS.Type.Machine = guest.Arch.Machines[0].Name
	}

	canonicalmachine, err := getCanonicalMachineName(caps, d.OS.Type.Arch, d.OS.Type.Type, d.OS.Type.Machine)
	if err != nil {
		return d, err
	}
	d.OS.Type.Machine = canonicalmachine
	return d, nil
}

func setCoreOSIgnition(domainDef *libvirtxml.Domain) error {
	ignitionKey := ignKey
	domainDef.QEMUCommandline = &libvirtxml.DomainQEMUCommandline{
		Args: []libvirtxml.DomainQEMUCommandlineArg{
			{
				Value: "-fw_cfg",
			},
			{
				Value: fmt.Sprintf("name=opt/com.coreos/config,file=%s", ignitionKey),
			},
		},
	}
	return nil
}

// note, source is not initialized
func newDefDisk(i int) libvirtxml.DomainDisk {
	return libvirtxml.DomainDisk{
		Device: "disk",
		Target: &libvirtxml.DomainDiskTarget{
			Bus: "virtio",
			Dev: fmt.Sprintf("vd%s", diskLetterForIndex(i)),
		},
		Driver: &libvirtxml.DomainDiskDriver{
			Name: "qemu",
			Type: "qcow2",
		},
	}
}

var diskLetters = []rune("abcdefghijklmnopqrstuvwxyz")

const oui = "05abcd"

// diskLetterForIndex return diskLetters for index
func diskLetterForIndex(i int) string {

	q := i / len(diskLetters)
	r := i % len(diskLetters)
	letter := diskLetters[r]

	if q == 0 {
		return fmt.Sprintf("%c", letter)
	}

	return fmt.Sprintf("%s%c", diskLetterForIndex(q-1), letter)
}
func randomWWN(strlen int) string {
	const chars = "abcdef0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return oui + string(result)
}

func setDisks(domainDef *libvirtxml.Domain, virConn *libvirt.Connect) error {
	disk := newDefDisk(0)
	log.Printf("[INFO] LookupStorageVolByKey")
	diskVolume, err := virConn.LookupStorageVolByKey(volumeKey)
	if err != nil {
		return fmt.Errorf("Can't retrieve volume %s", volumeKey)
	}
	log.Printf("[INFO] diskVolume")
	diskVolumeFile, err := diskVolume.GetPath()
	if err != nil {
		return fmt.Errorf("Error retrieving volume file: %s", err)
	}

	log.Printf("[INFO] DomainDiskSource")
	disk.Source = &libvirtxml.DomainDiskSource{
		File: &libvirtxml.DomainDiskSourceFile{
			File: diskVolumeFile,
		},
	}

	domainDef.Devices.Disks = append(domainDef.Devices.Disks, disk)

	return nil
}

// randomMACAddress returns a randomized MAC address
func randomMACAddress() (string, error) {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	// set local bit and unicast
	buf[0] = (buf[0] | 2) & 0xfe
	// Set the local bit
	buf[0] |= 2

	// avoid libvirt-reserved addresses
	if buf[0] == 0xfe {
		buf[0] = 0xee
	}

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		buf[0], buf[1], buf[2], buf[3], buf[4], buf[5]), nil
}

// Network interface used to expose a libvirt.Network
type Network interface {
	GetXMLDesc(flags libvirt.NetworkXMLFlags) (string, error)
}
func newDefNetworkfromLibvirt(network Network) (libvirtxml.Network, error) {
	networkXMLDesc, err := network.GetXMLDesc(0)
	if err != nil {
		return libvirtxml.Network{}, fmt.Errorf("Error retrieving libvirt domain XML description: %s", err)
	}
	networkDef := libvirtxml.Network{}
	err = xml.Unmarshal([]byte(networkXMLDesc), &networkDef)
	if err != nil {
		return libvirtxml.Network{}, fmt.Errorf("Error reading libvirt network XML description: %s", err)
	}
	return networkDef, nil
}

// HasDHCP checks if the network has a DHCP server managed by libvirt
func HasDHCP(net libvirtxml.Network) bool {
	if net.Forward != nil {
		if net.Forward.Mode == "nat" || net.Forward.Mode == "route" || net.Forward.Mode == "" {
			return true
		}
	}
	return false
}

// Tries to update first, if that fails, it will add it
func updateOrAddHost(n *libvirt.Network, ip, mac, name string) error {
	err := updateHost(n, ip, mac, name)
	if virErr, ok := err.(libvirt.Error); ok && virErr.Code == libvirt.ERR_OPERATION_INVALID && virErr.Domain == libvirt.FROM_NETWORK {
		return addHost(n, ip, mac, name)
	}
	return err
}
// Adds a new static host to the network
func addHost(n *libvirt.Network, ip, mac, name string) error {
	xmlDesc := getHostXMLDesc(ip, mac, name)
	log.Printf("Adding host with XML:\n%s", xmlDesc)
	return n.Update(libvirt.NETWORK_UPDATE_COMMAND_ADD_LAST, libvirt.NETWORK_SECTION_IP_DHCP_HOST, -1, xmlDesc, libvirt.NETWORK_UPDATE_AFFECT_CURRENT)
}

func getHostXMLDesc(ip, mac, name string) string {
	dd := libvirtxml.NetworkDHCPHost{
		IP:   ip,
		MAC:  mac,
		Name: name,
	}
	tmp := struct {
		XMLName xml.Name `xml:"host"`
		libvirtxml.NetworkDHCPHost
	}{xml.Name{}, dd}
	xml, err := xmlMarshallIndented(tmp)
	if err != nil {
		panic("could not marshall host")
	}
	return xml
}

// return an indented XML
func xmlMarshallIndented(b interface{}) (string, error) {
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	enc.Indent("  ", "    ")
	if err := enc.Encode(b); err != nil {
		return "", fmt.Errorf("could not marshall this:\n%s", spew.Sdump(b))
	}
	return buf.String(), nil
}

// Update a static host from the network
func updateHost(n *libvirt.Network, ip, mac, name string) error {
	xmlDesc := getHostXMLDesc(ip, mac, name)
	log.Printf("Updating host with XML:\n%s", xmlDesc)
	return n.Update(libvirt.NETWORK_UPDATE_COMMAND_MODIFY, libvirt.NETWORK_SECTION_IP_DHCP_HOST, -1, xmlDesc, libvirt.NETWORK_UPDATE_AFFECT_CURRENT)
}

func setNetworkInterfaces(domainDef *libvirtxml.Domain,
	virConn *libvirt.Connect, partialNetIfaces map[string]*pendingMapping,
	waitForLeases *[]*libvirtxml.DomainInterface) error {
	for i := 0; i < 1; i++ {
		//prefix := fmt.Sprintf("network_interface.%d", i)

		netIface := libvirtxml.DomainInterface{
			Model: &libvirtxml.DomainInterfaceModel{
				Type: "virtio",
			},
		}

		// calculate the MAC address
		var err error
		mac, err := randomMACAddress()
		if err != nil {
			return fmt.Errorf("Error generating mac address: %s", err)
		}
		netIface.MAC = &libvirtxml.DomainInterfaceMAC{
			Address: mac,
		}

		// this is not passed to libvirt, but used by waitForAddress
		//if waitForLease, ok := d.GetOk(prefix + ".wait_for_lease"); ok {
		//	if waitForLease.(bool) {
		//		*waitForLeases = append(*waitForLeases, &netIface)
		//	}
		//}

		// connect to the interface to the network... first, look for the network
		//if n, ok := d.GetOk(prefix + ".network_name"); ok {
		//	// when using a "network_name" we do not try to do anything: we just
		//	// connect to that network
		//	netIface.Source = &libvirtxml.DomainInterfaceSource{
		//		Network: &libvirtxml.DomainInterfaceSourceNetwork{
		//			Network: n.(string),
		//		},
		//	}
		//}
		networkUUID := networkUUID
		// when using a "network_id" we are referring to a "network resource"
		// we have defined somewhere else...
		network, err := virConn.LookupNetworkByUUIDString(networkUUID)
		if err != nil {
			return fmt.Errorf("Can't retrieve network ID %s", networkUUID)
		}
		defer network.Free()

		networkName, err := network.GetName()
		if err != nil {
			return fmt.Errorf("Error retrieving network name: %s", err)
		}
		networkDef, err := newDefNetworkfromLibvirt(network)

		// connect to the interface to the network... first, look for the network
		if networkInterfaceName != ""{
			// when using a "network_name" we do not try to do anything: we just
			// connect to that network
			netIface.Source = &libvirtxml.DomainInterfaceSource{
				Network: &libvirtxml.DomainInterfaceSourceNetwork{
					Network: networkInterfaceName,
				},
			}
		} else if HasDHCP(networkDef) {
			hostname := networkInterfaceHostname
			//if addresses, ok := d.GetOk(prefix + ".addresses"); ok {
			if true {
				// some IP(s) provided
				address := networkInterfaceAddress
				ip := net.ParseIP(address)
				if ip == nil {
					return fmt.Errorf("Could not parse addresses '%s'", address)
				}

				log.Printf("[INFO] Adding IP/MAC/host=%s/%s/%s to %s", ip.String(), mac, hostname, networkName)
				if err := updateOrAddHost(network, ip.String(), mac, hostname); err != nil {
					return err
				}
			} else {
				// no IPs provided: if the hostname has been provided, wait until we get an IP
				wait := false
				for _, iface := range *waitForLeases {
					if iface == &netIface {
						wait = true
						break
					}
				}
				if !wait {
					return fmt.Errorf("Cannot map '%s': we are not waiting for DHCP lease and no IP has been provided", hostname)
				}
				// the resource specifies a hostname but not an IP, so we must wait until we
				// have a valid lease and then read the IP we have been assigned, so we can
				// do the mapping
				log.Printf("[DEBUG] Do not have an IP for '%s' yet: will wait until DHCP provides one...", hostname)
				partialNetIfaces[strings.ToUpper(mac)] = &pendingMapping{
					mac:      strings.ToUpper(mac),
					hostname: hostname,
					network:  network,
				}
			}
		}

		netIface.Source = &libvirtxml.DomainInterfaceSource{
			Network: &libvirtxml.DomainInterfaceSourceNetwork{
				Network: networkName,
			},
		}
		domainDef.Devices.Interfaces = append(domainDef.Devices.Interfaces, netIface)
	}

	return nil
}

// Config struct for the libvirt-provider
type Config struct {
	URI string
}

// Client libvirt, generate libvirt client given URI
func (c *Config) Client() (*Client, error) {
	libvirtClient, err := libvirt.NewConnect(c.URI)
	if err != nil {
		return nil, err
	}
	log.Println("[INFO] Created libvirt client")

	client := &Client{
		libvirt:     libvirtClient,
	}

	return client, nil
}

func createDomain() error {
	log.Printf("[DEBUG] Create resource libvirt_domain")

	config := &Config{
		URI: uri,
	}
	client, err := config.Client(); if err != nil {
		return fmt.Errorf("Failed to build libvirt client: %s", err)
	}
	virConn := client.libvirt

	domainDef, err := newDomainDefForConnection(virConn)
	if err != nil {
		return fmt.Errorf("Failed to newDomainDefForConnection: %s", err)
	}

	domainDef.Name = domainName

	//if cpuMode, ok := d.GetOk("cpu.mode"); ok {
	//	domainDef.CPU = &libvirtxml.DomainCPU{
	//		Mode: cpuMode.(string),
	//	}
	//}

	domainDef.Memory = &libvirtxml.DomainMemory{
		Value: uint(domainMemory),
		Unit:  "MiB",
	}
	domainDef.VCPU = &libvirtxml.DomainVCPU{
		Value: domainVcpu,
	}

	//domainDef.OS.Kernel = d.Get("kernel").(string)
	//domainDef.OS.Initrd = d.Get("initrd").(string)
	//domainDef.OS.Type.Arch = d.Get("arch").(string)
	//domainDef.OS.Type.Machine = d.Get("machine").(string)
	//domainDef.Devices.Emulator = d.Get("emulator").(string)

	//arch, err := getHostArchitecture(virConn)
	//if err != nil {
	//	return fmt.Errorf("Error retrieving host architecture: %s", err)
	//}

	//if err := setGraphics(d, &domainDef, arch); err != nil {
	//	return err
	//}

	//setConsoles(d, &domainDef)
	//setCmdlineArgs(d, &domainDef)
	//setFirmware(d, &domainDef)
	//setBootDevices(d, &domainDef)

	log.Printf("[INFO] setCoreOSIgnition")
	if err := setCoreOSIgnition(&domainDef); err != nil {
		return err
	}

	log.Printf("[INFO] setDisks")
	if err := setDisks(&domainDef, virConn); err != nil {
		log.Printf("[INFO] Failed to setDisks log")
		return fmt.Errorf("Failed to setDisks: %s", err)
	}

	//if err := setFilesystems(d, &domainDef); err != nil {
	//	return err
	//}

	log.Printf("[INFO] setNetworkInterfaces")
	var waitForLeases []*libvirtxml.DomainInterface
	//partialNetIfaces := make(map[string]*pendingMapping, d.Get("network_interface.#").(int))
	partialNetIfaces := make(map[string]*pendingMapping, 1)

	if err := setNetworkInterfaces(&domainDef, virConn, partialNetIfaces, &waitForLeases); err != nil {
		return err
	}

	connectURI, err := virConn.GetURI()
	if err != nil {
		return fmt.Errorf("Error retrieving libvirt connection URI: %s", err)
	}
	log.Printf("[INFO] Creating libvirt domain at %s", connectURI)

	data, err := xmlMarshallIndented(domainDef)
	if err != nil {
		return fmt.Errorf("Error serializing libvirt domain: %s", err)
	}

	log.Printf("[DEBUG] Creating libvirt domain with XML:\n%s", data)

	domain, err := virConn.DomainDefineXML(data)
	if err != nil {
		return fmt.Errorf("Error defining libvirt domain: %s", err)
	}

	err = domain.SetAutostart(autostart)

	err = domain.Create()
	if err != nil {
		return fmt.Errorf("Error creating libvirt domain: %s", err)
	}
	defer domain.Free()

	id, err := domain.GetUUIDString()
	if err != nil {
		return fmt.Errorf("Error retrieving libvirt domain id: %s", err)
	}

	log.Printf("[INFO] Domain ID: %s", id)

	//if len(waitForLeases) > 0 {
	//	err = domainWaitForLeases(domain, waitForLeases, d.Timeout(schema.TimeoutCreate),
	//		domainDef, virConn)
	//	if err != nil {
	//		return err
	//	}
	//}

	//err = resourceLibvirtDomainRead(d, meta)
	//if err != nil {
	//	return err
	//}
	//
	//// we must read devices again in order to set some missing ip/MAC/host mappings
	//for i := 0; i < d.Get("network_interface.#").(int); i++ {
	//	prefix := fmt.Sprintf("network_interface.%d", i)
	//
	//	mac := strings.ToUpper(d.Get(prefix + ".mac").(string))
	//
	//	// if we were waiting for an IP address for this MAC, go ahead.
	//	if pending, ok := partialNetIfaces[mac]; ok {
	//		// we should have the address now
	//		addressesI, ok := d.GetOk(prefix + ".addresses")
	//		if !ok {
	//			return fmt.Errorf("Did not obtain the IP address for MAC=%s", mac)
	//		}
	//		for _, addressI := range addressesI.([]interface{}) {
	//			address := addressI.(string)
	//			log.Printf("[INFO] Finally adding IP/MAC/host=%s/%s/%s", address, mac, pending.hostname)
	//			updateOrAddHost(pending.network, address, mac, pending.hostname)
	//			if err != nil {
	//				return fmt.Errorf("Could not add IP/MAC/host=%s/%s/%s: %s", address, mac, pending.hostname, err)
	//			}
	//		}
	//	}
	//}
	//
	//destroyDomainByUserRequest(d, domain)
	return nil
}

func main() {
	if err := resourceLibvirtVolumeCreate(); err != nil {
		log.Fatalf("Main: %s", err)
	}
	if err := createDomain(); err != nil {
		log.Fatalf("Main: %s", err)
	}
}
