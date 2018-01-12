package main

import (
	"context"

	pbgd "github.com/brotherlogic/godiscogs"
	pb "github.com/brotherlogic/recordcollection/proto"
)

func add(ctx context.Context, client pb.RecordCollectionServiceClient, id int32, cost int32, folder int32) (*pb.AddRecordResponse, error) {
	return client.AddRecord(ctx, &pb.AddRecordRequest{ToAdd: &pb.Record{
		Release:  &pbgd.Release{Id: id},
		Metadata: &pb.ReleaseMetadata{Cost: cost, GoalFolder: folder},
	}})
}
