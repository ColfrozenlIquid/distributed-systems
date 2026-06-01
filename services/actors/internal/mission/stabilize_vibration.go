package mission

import (
	pb "actors/gen"
	proto "actors/gen"
	"context"
)

type StabilizeVibrationMission struct {
	simulation pb.WorldServiceClient
}

func (s *StabilizeVibrationMission) RegisterMission(ctx context.Context, req *proto.AssignMissionRequest, ch chan<- *proto.AssignMissionResponse) error {
	panic("unimplemented")
}

func NewStabilizeVibrationMission(req *pb.AssignMissionRequest, ch chan<- *pb.AssignMissionResponse, simulation pb.WorldServiceClient) *StabilizeVibrationMission {
	return &StabilizeVibrationMission{}
}
