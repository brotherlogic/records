package main

import (
	"context"
	"fmt"

	gd "github.com/brotherlogic/godiscogs"
	pbrc "github.com/brotherlogic/recordcollection/proto"
)

func listUncategorized(ctx context.Context, client pbrc.RecordCollectionServiceClient) error {

	releases, err := client.GetRecords(ctx, &pbrc.GetRecordsRequest{Filter: &pbrc.Record{Release: &gd.Release{FolderId: 1}}})

	if err != nil {
		return err
	}

	if len(releases.GetRecords()) == 0 {
		fmt.Printf("No uncategorized releases!\n")
	}

	for _, release := range releases.GetRecords() {
		fmt.Printf("%v: %v - %v\n", release.GetRelease().Id, gd.GetReleaseArtist(release.GetRelease()), release.GetRelease().Title)
	}

	return nil
}
