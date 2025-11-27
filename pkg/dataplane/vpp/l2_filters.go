package vpp

type L2Filter struct {
	Name      string
	EtherType uint16
	Protocol  uint8
}

// Pre-configured L2 filters per 3GPP TS 29.244 Application ID approach, this is kind of a hack because PFCP doesn't define lower layer protocols, but for TR-459 BNG CUPs, we need to program rules to punt ARP/v6 ND/PPP packets
// While we shouldn't really ever need to punt ARP/ND because the user plane should be responsible for this, we might still want to do something with the PPP layer on the control plane
var l2FilterRegistry = map[string]*L2Filter{
	"ARP": {
		Name:      "ARP",
		EtherType: 0x0806,
	},
	"PPPOE_DISCOVERY": {
		Name:      "PPPoE Discovery",
		EtherType: 0x8863,
	},
	"PPPOE_SESSION": {
		Name:      "PPPoE Session",
		EtherType: 0x8864,
	},
	"LLDP": {
		Name:      "LLDP",
		EtherType: 0x88cc,
	},
	"DOT1Q": {
		Name:      "802.1Q VLAN",
		EtherType: 0x8100,
	},
	"IPV6": {
		Name:      "IPv6",
		EtherType: 0x86dd,
	},
}

func GetL2Filter(applicationID string) (*L2Filter, bool) {
	filter, ok := l2FilterRegistry[applicationID]
	return filter, ok
}
