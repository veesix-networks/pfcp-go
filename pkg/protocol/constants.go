package protocol

const (
	Version1 uint8 = 1
	UDPPort        = 8805
)

const (
	MsgTypeHeartbeatRequest           uint8 = 1
	MsgTypeHeartbeatResponse          uint8 = 2
	MsgTypePFDManagementRequest       uint8 = 3
	MsgTypePFDManagementResponse      uint8 = 4
	MsgTypeAssociationSetupRequest    uint8 = 5
	MsgTypeAssociationSetupResponse   uint8 = 6
	MsgTypeAssociationUpdateRequest   uint8 = 7
	MsgTypeAssociationUpdateResponse  uint8 = 8
	MsgTypeAssociationReleaseRequest  uint8 = 9
	MsgTypeAssociationReleaseResponse uint8 = 10
	MsgTypeVersionNotSupported        uint8 = 11
	MsgTypeNodeReportRequest          uint8 = 12
	MsgTypeNodeReportResponse         uint8 = 13
	MsgTypeSessionSetDeletionRequest  uint8 = 14
	MsgTypeSessionSetDeletionResponse uint8 = 15
)

const (
	MsgTypeSessionEstablishmentRequest  uint8 = 50
	MsgTypeSessionEstablishmentResponse uint8 = 51
	MsgTypeSessionModificationRequest   uint8 = 52
	MsgTypeSessionModificationResponse  uint8 = 53
	MsgTypeSessionDeletionRequest       uint8 = 54
	MsgTypeSessionDeletionResponse      uint8 = 55
	MsgTypeSessionReportRequest         uint8 = 56
	MsgTypeSessionReportResponse        uint8 = 57
)

const (
	IETypeCause                 uint16 = 19
	IETypeSourceInterface       uint16 = 20
	IETypeNetworkInstance       uint16 = 22
	IETypeSDFFilter             uint16 = 23
	IETypeApplicationID         uint16 = 24
	IETypeGateStatus            uint16 = 25
	IETypeMBR                   uint16 = 26
	IETypeGBR                   uint16 = 27
	IETypePrecedence            uint16 = 29
	IETypeReportingTriggers     uint16 = 37
	IETypeDestinationInterface  uint16 = 42
	IETypeApplyAction           uint16 = 44
	IETypeNodeID                uint16 = 60
	IETypeMeasurementMethod     uint16 = 62
	IETypeURR_ID                uint16 = 81
	IETypeUE_IPAddress          uint16 = 93
	IETypeOuterHeaderRemoval    uint16 = 95
	IETypeRecoveryTimeStamp     uint16 = 96
	IETypeFAR_ID                uint16 = 108
	IETypeQER_ID                uint16 = 109

	IETypePDI                  uint16 = 2
	IETypeForwardingParameters uint16 = 4
	IETypeCreatePDR            uint16 = 1
	IETypeCreateFAR            uint16 = 3
	IETypeCreateQER            uint16 = 7
	IETypeCreateURR            uint16 = 6
	IETypePDR_ID               uint16 = 56
	IETypeFlowDescription      uint16 = 106
)

const (
	SourceInterfaceAccess      uint8 = 0
	SourceInterfaceCore        uint8 = 1
	SourceInterfaceSGiLAN      uint8 = 2
	SourceInterfaceCPFunction  uint8 = 3
)

const (
	DestinationInterfaceAccess      uint8 = 0
	DestinationInterfaceCore        uint8 = 1
	DestinationInterfaceSGiLAN      uint8 = 2
	DestinationInterfaceCPFunction  uint8 = 3
)

const (
	ApplyActionDrop      uint8 = 0x01
	ApplyActionForward   uint8 = 0x02
	ApplyActionBuffer    uint8 = 0x04
	ApplyActionNotify    uint8 = 0x08
	ApplyActionDuplicate uint8 = 0x10
)

const (
	CauseRequestAccepted                uint8 = 1
	CauseRequestRejected                uint8 = 64
	CauseSessionContextNotFound         uint8 = 65
	CauseMandatoryIEMissing             uint8 = 66
	CauseConditionalIEMissing           uint8 = 67
	CauseInvalidLength                  uint8 = 68
	CauseMandatoryIEIncorrect           uint8 = 69
	CauseInvalidForwardingPolicy        uint8 = 70
	CauseInvalidFTEID                   uint8 = 71
	CauseNoEstablishedPFCPAssociation   uint8 = 72
	CauseRuleCreationModificationFailure uint8 = 73
	CausePFCPEntityInCongestion         uint8 = 74
	CauseNoResourcesAvailable           uint8 = 75
	CauseServiceNotSupported            uint8 = 76
	CauseSystemFailure                  uint8 = 77
)

const (
	MeasurementMethodDuration uint8 = 0x01
	MeasurementMethodVolume   uint8 = 0x02
	MeasurementMethodEvent    uint8 = 0x04
)

const (
	ReportingTriggerPeriodicReporting         uint32 = 0x00000001
	ReportingTriggerVolumeThreshold           uint32 = 0x00000002
	ReportingTriggerTimeThreshold             uint32 = 0x00000004
	ReportingTriggerQuotaHoldingTime          uint32 = 0x00000008
	ReportingTriggerStartOfTraffic            uint32 = 0x00000010
	ReportingTriggerStopOfTraffic             uint32 = 0x00000020
	ReportingTriggerDroppedDLTrafficThreshold uint32 = 0x00000040
	ReportingTriggerLinkedUsageReporting      uint32 = 0x00000080
	ReportingTriggerVolumeQuota               uint32 = 0x00000100
	ReportingTriggerTimeQuota                 uint32 = 0x00000200
	ReportingTriggerEnvelopeClosure           uint32 = 0x00000400
)
