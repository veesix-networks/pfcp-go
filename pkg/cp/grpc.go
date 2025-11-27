package cp

import (
	"context"
	"fmt"
	"net"

	pb "github.com/veesix-networks/pfcp-go/api/pfcp/v1"
)

type GRPCServer struct {
	pb.UnimplementedControlPlaneServer
	cp *CPFunction
}

func NewGRPCServer(cp *CPFunction) *GRPCServer {
	return &GRPCServer{cp: cp}
}

func (s *GRPCServer) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.CreateSessionResponse, error) {
	pdrs := make([]*PDR, len(req.Pdrs))
	for i, pdr := range req.Pdrs {
		var ueIP net.IP
		if pdr.Pdi.UeIpAddress != "" {
			ueIP = net.ParseIP(pdr.Pdi.UeIpAddress)
		}

		pdrs[i] = &PDR{
			ID:         uint16(pdr.Id),
			Precedence: pdr.Precedence,
			PDI: &PacketDetectionInfo{
				SourceInterface: uint8(pdr.Pdi.SourceInterface),
				SDFFilter:       pdr.Pdi.SdfFilter,
				UE_IPAddress:    ueIP,
				NetworkInstance: pdr.Pdi.NetworkInstance,
				ApplicationID:   pdr.Pdi.ApplicationId,
			},
			FAR_ID: pdr.FarId,
		}
	}

	fars := make([]*FAR, len(req.Fars))
	for i, far := range req.Fars {
		fars[i] = &FAR{
			ID:          far.Id,
			ApplyAction: uint8(far.ApplyAction),
			ForwardingParameters: &ForwardingParams{
				DestinationInterface: uint8(far.ForwardingParams.DestinationInterface),
				NetworkInstance:      far.ForwardingParams.NetworkInstance,
			},
		}
	}

	var qers []*QER
	if len(req.Qers) > 0 {
		qers = make([]*QER, len(req.Qers))
		for i, qer := range req.Qers {
			qers[i] = &QER{
				ID:         qer.Id,
				GateStatus: uint8(qer.GateStatus),
				MBR_UL:     qer.MbrUplink,
				MBR_DL:     qer.MbrDownlink,
			}
		}
	}

	var urrs []*URR
	if len(req.Urrs) > 0 {
		urrs = make([]*URR, len(req.Urrs))
		for i, urr := range req.Urrs {
			urrs[i] = &URR{
				ID:                urr.Id,
				MeasurementMethod: uint8(urr.MeasurementMethod),
			}
		}
	}

	seid, err := s.cp.CreateSession(req.NodeId, pdrs, fars, qers, urrs)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	fmt.Printf("gRPC: Session created SEID=%d for node %s\n", seid, req.NodeId)

	return &pb.CreateSessionResponse{Seid: seid}, nil
}

func (s *GRPCServer) ModifySession(ctx context.Context, req *pb.ModifySessionRequest) (*pb.ModifySessionResponse, error) {
	return nil, fmt.Errorf("ModifySession not implemented yet")
}

func (s *GRPCServer) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.DeleteSessionResponse, error) {
	err := s.cp.DeleteSession(req.Seid)
	if err != nil {
		return nil, fmt.Errorf("delete session: %w", err)
	}

	fmt.Printf("gRPC: Session deleted SEID=%d\n", req.Seid)

	return &pb.DeleteSessionResponse{Success: true}, nil
}

func (s *GRPCServer) ListAssociations(ctx context.Context, req *pb.ListAssociationsRequest) (*pb.ListAssociationsResponse, error) {
	s.cp.mu.RLock()
	defer s.cp.mu.RUnlock()

	associations := make([]*pb.Association, 0, len(s.cp.associations))
	for nodeID, assoc := range s.cp.associations {
		associations = append(associations, &pb.Association{
			NodeId:         nodeID,
			RemoteAddr:     assoc.RemoteAddr.String(),
			EstablishedAt:  assoc.EstablishedAt.Unix(),
		})
	}

	return &pb.ListAssociationsResponse{Associations: associations}, nil
}
