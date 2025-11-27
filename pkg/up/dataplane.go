package up

type Dataplane interface {
	InstallPDR(seid uint64, pdr *PDR) error
	RemovePDR(seid uint64, pdrID uint16) error
	InstallFAR(seid uint64, far *FAR) error
	RemoveFAR(seid uint64, farID uint32) error
	InstallQER(seid uint64, qer *QER) error
	RemoveQER(seid uint64, qerID uint32) error
	InstallURR(seid uint64, urr *URR) error
	RemoveURR(seid uint64, urrID uint32) error
	DeleteSession(seid uint64) error
}
