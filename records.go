package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/brotherlogic/discogssyncer/server"
	pbdi "github.com/brotherlogic/discovery/proto"
	pbd "github.com/brotherlogic/godiscogs"
	"github.com/brotherlogic/goserver/utils"
	pbo "github.com/brotherlogic/recordsorganiser/proto"
)

func getIP(servername string) (string, int) {
	conn, _ := grpc.Dial(utils.RegistryIP+":"+strconv.Itoa(utils.RegistryPort), grpc.WithInsecure())
	defer conn.Close()

	registry := pbdi.NewDiscoveryServiceClient(conn)
	entry := pbdi.RegistryEntry{Name: servername}
	r, err := registry.Discover(context.Background(), &entry)
	if err != nil {
		return "", -1
	}
	return r.Ip, int(r.Port)
}

func listFolder(ID int32) {
	dServer, dPort := getIP("discogssyncer")
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)
	releases, err := dClient.GetReleasesInFolder(context.Background(), &pb.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: ID}}})

	if err != nil {
		panic(err)
	}

	for _, release := range releases.Records {
		fmt.Printf("%v: %v - %v\n", release.GetRelease().Id, pbd.GetReleaseArtist(release.GetRelease()), release.GetRelease().Title)
	}
}

func addLocation(name string, units int, folders string) {
	location := &pbo.Location{Name: name, Units: int32(units)}
	for _, folder := range strings.Split(folders, ",") {
		folderID, _ := strconv.Atoi(folder)
		location.FolderIds = append(location.FolderIds, int32(folderID))
	}

	//Move the previous record down to uncategorized
	dServer, dPort := getIP("recordsorganiser")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer dConn.Close()
	dClient := pbo.NewOrganiserServiceClient(dConn)
	log.Printf("Sending: %v", location)
	newLocation, err := dClient.AddLocation(context.Background(), location)
	log.Printf("New Location = %v (%v)", newLocation, err)
}

func addRecord(id int) {
	dServer, dPort := getIP("discogssyncer")

	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	release := &pbd.Release{Id: int32(id)}
	folderAdd := &pb.ReleaseMove{Release: release, NewFolderId: int32(812802)}

	_, err = dClient.AddToFolder(context.Background(), folderAdd)
	if err != nil {
		panic(err)
	}
}

func getLocation(name string, slot int32, timestamp int64) {
	//Move the previous record down to uncategorized
	server, port := getIP("recordsorganiser")
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	locationQuery := &pbo.Location{Name: name, Timestamp: timestamp}
	location, err := client.GetLocation(context.Background(), locationQuery)

	if err != nil {
		panic(err)
	}

	fmt.Printf("%v sorted %v\n", location.Name, location.Sort)

	if err != nil {
		panic(err)
	}
	dServer, dPort := getIP("discogssyncer")
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	var relMap map[int32]*pbd.Release
	relMap = make(map[int32]*pbd.Release)

	for _, folderID := range location.FolderIds {
		releases, err := dClient.GetReleasesInFolder(context.Background(), &pb.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: folderID}}})

		if err != nil {
			log.Printf("Cannot retrieve folder %v", folderID)
			panic(err)
		}

		for _, rel := range releases.Records {
			relMap[rel.GetRelease().Id] = rel.GetRelease()
		}
	}
	width := int32(0)
	for _, release := range location.ReleasesLocation {
		if release.Slot == slot {
			fullRelease, err := dClient.GetSingleRelease(context.Background(), &pbd.Release{Id: release.ReleaseId})
			if err == nil {
				width += fullRelease.FormatQuantity
				fmt.Printf("%v. [%v] %v - %v (%v)\n", release.Index, width, pbd.GetReleaseArtist(fullRelease), fullRelease.Title, fullRelease.Id)
			}
		}
	}

}

func getRelease(id int32) (*pbd.Release, *pb.ReleaseMetadata) {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		return nil, nil
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	releaseRequest := &pbd.Release{Id: id}
	rel, _ := dClient.GetSingleRelease(context.Background(), releaseRequest)
	meta, _ := dClient.GetMetadata(context.Background(), rel)
	return rel, meta
}

func getAllReleases() []*pbd.Release {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		log.Printf("FAIL %v", err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	releases, _ := dClient.GetCollection(context.Background(), &pb.Empty{})
	return releases.Releases
}

func prettyPrintRelease(id int32) string {
	rel, _ := getRelease(id)
	if rel != nil {
		return pbd.GetReleaseArtist(rel) + " - " + rel.Title
	}
	if id == 0 {
		return "---------------"
	}
	return strconv.Itoa(int(id))
}

func listFolders() {
	fmt.Printf("Folders:\n")
	server, port := getIP("recordsorganiser")
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	org, err := client.GetOrganisation(context.Background(), &pbo.Empty{})

	if err != nil {
		panic(err)
	}

	for _, location := range org.Locations {
		fmt.Printf("%v\n", location.Name)
	}
}

func organise(doSlotMoves bool) {
	server, port := getIP("recordsorganiser")
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	log.Printf("Request re-org from %v:%v", server, port)
	moves, err := client.Organise(context.Background(), &pbo.Empty{})

	if err != nil {
		panic(err)
	}

	fmt.Printf("Org from %v to %v\n", moves.StartTimestamp, moves.EndTimestamp)

	if len(moves.Moves) == 0 {
		fmt.Printf("No Moves needed\n")
	}

	for _, move := range moves.Moves {
		if doSlotMoves || !move.SlotMove {
			printMove(move)
		}
	}
}

func printMove(move *pbo.LocationMove) {
	fmt.Printf("----------------")
	if move.Old == nil {
		fmt.Printf("Add to slot %v\n", move.New.Slot)
		fmt.Printf("%v\n*%v*\n%v\n", prettyPrintRelease(move.New.BeforeReleaseId), prettyPrintRelease(move.New.ReleaseId), prettyPrintRelease(move.New.AfterReleaseId))
	} else if move.New == nil {
		fmt.Printf("Remove from slot %v\n", move.Old.Slot)
		fmt.Printf("%v\n*%v*\n%v\n", prettyPrintRelease(move.Old.BeforeReleaseId), prettyPrintRelease(move.Old.ReleaseId), prettyPrintRelease(move.Old.AfterReleaseId))
	} else {
		fmt.Printf("Move from slot %v to slot %v\n", move.Old.Slot, move.New.Slot)
		fmt.Printf("%v\n*%v*\n%v\n", prettyPrintRelease(move.Old.BeforeReleaseId), prettyPrintRelease(move.Old.ReleaseId), prettyPrintRelease(move.Old.AfterReleaseId))
		fmt.Printf("to\n")
		fmt.Printf("%v\n*%v*\n%v\n", prettyPrintRelease(move.New.BeforeReleaseId), prettyPrintRelease(move.New.ReleaseId), prettyPrintRelease(move.New.AfterReleaseId))
	}
}

func updateLocation(loc *pbo.Location) {
	server, port := getIP("recordsorganiser")
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	r, err := client.UpdateLocation(context.Background(), loc)
	fmt.Printf("From %v got %v\n", loc, r)
}

func printDiff(diffRequest *pbo.DiffRequest) {
	server, port := getIP("recordsorganiser")
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	moves, err := client.Diff(context.Background(), diffRequest)

	if err != nil {
		panic(err)
	}

	for _, move := range moves.Moves {
		printMove(move)
		fmt.Printf("\n")
	}
}
func locate(id int) {
	server, port := getIP("recordsorganiser")
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	res, err := client.Locate(context.Background(), &pbd.Release{Id: int32(id)})

	if err != nil {
		log.Fatalf("Unable to locate %v: %v", id, err)
	}

	fmt.Printf("In %v, slot %v\n", res.Location.Name, res.Slot)
	if res.Before != nil {
		fmt.Printf("Before: %v - %v (%v)\n", pbd.GetReleaseArtist(res.Before), res.Before.Title, res.Before.Id)
	}
	if res.After != nil {
		fmt.Printf("After:  %v - %v (%v)\n", pbd.GetReleaseArtist(res.After), res.After.Title, res.After.Id)
	}
}

func moveToPile(id int) {
	log.Printf("Moving")
	dServer, dPort := getIP("discogssyncer")
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	releases, err := dClient.GetReleasesInFolder(context.Background(), &pb.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: 1}}})
	if err != nil {
		log.Fatalf("Fatal error in getting releases: %v", err)
	}

	for _, release := range releases.Records {
		if release.GetRelease().Id == int32(id) {
			move := &pb.ReleaseMove{Release: &pbd.Release{Id: int32(id), FolderId: 1, InstanceId: release.GetRelease().InstanceId}, NewFolderId: int32(812802)}
			_, err = dClient.MoveToFolder(context.Background(), move)
			log.Printf("MOVED %v from %v", move, release)
			if err != nil {
				panic(err)
			}
		}
	}
}

func move(id int, folderID int) {
	log.Printf("Moving")
	dServer, dPort := getIP("discogssyncer")
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	r, _ := dClient.GetSingleRelease(context.Background(), &pbd.Release{Id: int32(id)})
	releases, err := dClient.GetReleasesInFolder(context.Background(), &pb.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: r.FolderId}}})
	if err != nil {
		log.Fatalf("Fatal error in getting releases: %v", err)
	}

	for _, release := range releases.Records {
		if release.GetRelease().Id == int32(id) {
			move := &pb.ReleaseMove{Release: &pbd.Release{Id: int32(id), FolderId: release.GetRelease().FolderId, InstanceId: release.GetRelease().InstanceId}, NewFolderId: int32(folderID)}
			_, err = dClient.MoveToFolder(context.Background(), move)
			log.Printf("MOVED %v from %v", move, release)
			if err != nil {
				panic(err)
			}
		}
	}
}

func collapseWantlist() {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)
	_, err = dClient.CollapseWantlist(context.Background(), &pb.Empty{})

	if err != nil {
		panic(err)
	}
}

func rebuildWantlist() {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)
	_, err = dClient.RebuildWantlist(context.Background(), &pb.Empty{})

	if err != nil {
		panic(err)
	}
}

func deleteWant(id int) {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)
	res, err := dClient.DeleteWant(context.Background(), &pb.Want{ReleaseId: int32(id)})

	if err != nil {
		panic(err)
	}

	for i, want := range res.Want {
		if want.Valued {
			fmt.Printf("%v. *** %v [%v]\n", i, prettyPrintRelease(want.ReleaseId), want.ReleaseId)
		} else {
			fmt.Printf("%v. %v [%v]\n", i, prettyPrintRelease(want.ReleaseId), want.ReleaseId)
		}
	}
}

func deleteAllWants() {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)
	res, err := dClient.GetWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		panic(err)
	}
	for _, want := range res.GetWant() {
		_, err := dClient.DeleteWant(context.Background(), &pb.Want{ReleaseId: int32(want.ReleaseId)})

		if err != nil {
			panic(err)
		}
	}
}

func printWantlist() {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	wants, err := dClient.GetWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		panic(err)
	}

	if len(wants.Want) == 0 {
		fmt.Printf("No wants recorded\n")
	}

	for i, want := range wants.Want {
		if want.Valued {
			fmt.Printf("%v. *** %v [%v]\n", i, prettyPrintRelease(want.ReleaseId), want.ReleaseId)
		} else {
			fmt.Printf("%v. %v [%v]\n", i, prettyPrintRelease(want.ReleaseId), want.ReleaseId)
		}
	}
}

func getSpend() (int, []*pb.MetadataUpdate) {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	//Two month rolling average
	quarter := time.Now().YearDay()/91 + 1
	ys := time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	log.Printf("DATE = %v", ys)
	start := ys.AddDate(0, 0, 91*(quarter-1))
	log.Printf("NOW %v from %v -> %v", start, 91*(quarter-1), time.Now().YearDay()/91)
	end := ys.AddDate(0, 0, 91*quarter)
	req := &pb.SpendRequest{Lower: start.Unix(), Upper: end.Unix()}
	log.Printf("Spend Request: %v", req)
	spend, err := dClient.GetSpend(context.Background(), req)
	if err != nil {
		panic(err)
	}
	return int(spend.TotalSpend), spend.GetSpends()
}

func setWant(id int, wantValue bool) {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	want := &pb.Want{ReleaseId: int32(id), Valued: wantValue}
	wantRet, err := dClient.EditWant(context.Background(), want)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v\n", wantRet)
}

func deleteLocation(name string) {
	//Move the previous record down to uncategorized
	server, port := getIP("recordsorganiser")
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	_, err = client.DeleteLocation(context.Background(), &pbo.Location{Name: name})
	if err != nil {
		panic(err)
	}
}

func printLow(name string, others bool) {
	//Move the previous record down to uncategorized
	server, port := getIP("recordsorganiser")
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	locationQuery := &pbo.Location{Name: name}
	location, err := client.GetLocation(context.Background(), locationQuery)
	if err != nil {
		log.Fatalf("Fatal error in getting location: %v", err)
	}

	dServer, dPort := getIP("discogssyncer")
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	var lowest []*pbd.Release
	lowestScore := 6
	for _, folderID := range location.FolderIds {
		log.Printf("Checking folder: %v", folderID)
		releases, err := dClient.GetReleasesInFolder(context.Background(), &pb.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: folderID}}})

		if err != nil {
			panic(err)
		}

		for _, release := range releases.Records {
			if int(release.GetRelease().Rating) < lowestScore {
				lowestScore = int(release.GetRelease().Rating)
				lowest = make([]*pbd.Release, 0)
				lowest = append(lowest, release.GetRelease())
			} else if int(release.GetRelease().Rating) == lowestScore {
				lowest = append(lowest, release.GetRelease())
			}
		}
	}

	for i, release := range lowest {
		_, meta := getRelease(release.Id)
		if !others || meta.Others {
			fmt.Printf("%v [%v]. %v\n", i, release.Id, prettyPrintRelease(release.Id))
		}
	}
}

func updateMeta(ID int, date string) {
	dServer, dPort := getIP("discogssyncer")
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	dateAdded, err := time.Parse("02/01/06", date)
	if err != nil {
		log.Fatalf("Failure to parse date: %v", err)
	}
	update := &pb.MetadataUpdate{Release: &pbd.Release{Id: int32(ID)}, Update: &pb.ReleaseMetadata{DateAdded: dateAdded.Unix()}}
	dClient.UpdateMetadata(context.Background(), update)
}

func sell(ID int) {
	dServer, dPort := getIP("discogssyncer")

	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	release, _ := getRelease(int32(ID))
	folderAdd := &pb.ReleaseMove{Release: release, NewFolderId: int32(488127)}

	_, err = dClient.MoveToFolder(context.Background(), folderAdd)
	if err != nil {
		panic(err)
	}

	_, err = dClient.Sell(context.Background(), release)
	if err != nil {
		panic(err)
	}
}

func printTidy(place string) {
	//Move the previous record down to uncategorized
	server, port := getIP("recordsorganiser")
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	locationQuery := &pbo.Location{Name: place, Timestamp: -1}
	location, err := client.GetLocation(context.Background(), locationQuery)

	if err != nil {
		panic(err)
	}

	//Print out potential infractions
	infractions, err := client.CleanLocation(context.Background(), location)
	if err != nil {
		panic(err)
	}

	if len(infractions.Entries) > 0 {
		fmt.Printf("Infractions:\n")
		for _, inf := range infractions.Entries {
			fmt.Printf("%v.%v\n", inf.Id, prettyPrintRelease(inf.Id))
		}
	}

	log.Printf("RUNNING GET QUOTA VIOLATIONS")
	violations, err := client.GetQuotaViolations(context.Background(), &pbo.Empty{})
	if err != nil {
		panic(err)
	}

	if len(violations.GetLocations()) > 0 {
		for _, loc := range violations.Locations {
			fmt.Printf("Quota Violation in folder %v\n", loc.Name)
		}
	}

	dServer, dPort := getIP("discogssyncer")
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	var relMap map[int32]*pbd.Release
	relMap = make(map[int32]*pbd.Release)

	for _, folderID := range location.FolderIds {
		releases, err := dClient.GetReleasesInFolder(context.Background(), &pb.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: folderID}}})

		if err != nil {
			log.Printf("Cannot retrieve folder %v", folderID)
			panic(err)
		}

		for _, rel := range releases.Records {
			relMap[rel.GetRelease().Id] = rel.GetRelease()
		}
	}

	bestRelease := make(map[int32]*pbd.Release)
	bestIndex := make(map[int32]int32)
	for _, release := range location.ReleasesLocation {
		fullRelease, err := dClient.GetSingleRelease(context.Background(), &pbd.Release{Id: release.ReleaseId})
		if err == nil {
			if _, ok := bestIndex[release.Slot]; ok {
				if bestIndex[release.Slot] < release.Index {
					bestIndex[release.Slot] = release.Index
					bestRelease[release.Slot] = fullRelease
				}
			} else {
				bestIndex[release.Slot] = release.Index
				bestRelease[release.Slot] = fullRelease
			}
		}
	}

	slotv := int32(1)
	for slotv > 0 {
		if _, ok := bestIndex[slotv]; ok {
			fmt.Printf("%v. %v - %v\n", slotv, pbd.GetReleaseArtist(bestRelease[slotv]), bestRelease[slotv].Title)
			slotv++
		} else {
			slotv = -1
		}
	}
}

func delete(instance int) {
	dServer, dPort := getIP("discogssyncer")
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	if instance == 0 {
		found := true
		for found {
			r, errin := dClient.GetSingleRelease(context.Background(), &pbd.Release{Id: int32(0)})
			if errin != nil {
				log.Fatalf("Get FAIL: %v", err)
			}
			log.Printf("DELETING %v", r.InstanceId)
			if r.InstanceId > 0 {
				_, errin = dClient.DeleteInstance(context.Background(), &pbd.Release{InstanceId: r.InstanceId})
				if errin != nil {
					log.Fatalf("DELETE FAIL: %v", errin)
				}
			} else {
				found = false
			}
		}
	} else {
		_, err = dClient.DeleteInstance(context.Background(), &pbd.Release{InstanceId: int32(instance)})
		if err != nil {
			log.Printf("DELETE FAIL: %v", err)
		}
	}
}

func oldmain() {
	addFlags := flag.NewFlagSet("AddRecord", flag.ExitOnError)
	var id = addFlags.Int("id", 0, "ID of record to add")

	addLocationFlags := flag.NewFlagSet("AddLocation", flag.ExitOnError)
	var name = addLocationFlags.String("name", "", "The name of the new location")
	var units = addLocationFlags.Int("units", 0, "The number of units in the location")
	var folderIds = addLocationFlags.String("folders", "", "The list of folder IDs")

	getLocationFlags := flag.NewFlagSet("GetLocation", flag.ExitOnError)
	var getName = getLocationFlags.String("name", "", "The name of the location to get")
	var slot = getLocationFlags.Int("slot", 1, "The slot to retrieve from")
	var timestamp = getLocationFlags.Int64("time", -1, "The timestamp to retrieve")

	moveFlags := flag.NewFlagSet("Move", flag.ExitOnError)
	var idToMoveToFolder = moveFlags.Int("id", 0, "Id of record to move")
	var folderID = moveFlags.Int("folder", 0, "Id of folder to move to")

	locateFlags := flag.NewFlagSet("Locate", flag.ExitOnError)
	var idToLocate = locateFlags.Int("id", 0, "Id of record to locate")

	updateLocationFlags := flag.NewFlagSet("UpdateLocation", flag.ContinueOnError)
	var nameToUpdate = updateLocationFlags.String("name", "", "Name of the location to update")
	var sort = updateLocationFlags.String("sort", "", "Sorting method of the location")
	var updateFolders = updateLocationFlags.String("folders", "", "Folders to add")
	var numSlots = updateLocationFlags.Int("slots", -1, "The number of slots to update to")
	var formatexp = updateLocationFlags.String("format", "", "The format test")
	var unexpectedlabel = updateLocationFlags.String("unlabel", "", "The unexpected label test")
	var quotaNum = updateLocationFlags.Int("quota", -1, "The quota number")

	investigateFlags := flag.NewFlagSet("investigate", flag.ExitOnError)
	var investigateID = investigateFlags.Int("id", 0, "Id of release to investigate")
	var investigateYear = investigateFlags.Int("year", 0, "Year of releases to list")
	var deepInvestigate = investigateFlags.Bool("deep", false, "Do a deep search for the release")

	diffFlags := flag.NewFlagSet("diff", flag.ExitOnError)
	var diffSlot = diffFlags.Int("slot", 0, "The slot to check")
	var diffName = diffFlags.String("name", "", "The folder to check")

	lowFlags := flag.NewFlagSet("low", flag.ExitOnError)
	var lowFolderName = lowFlags.String("name", "", "Name of the folder to check")
	var showOthersOnly = lowFlags.Bool("others", false, "Show only records that we have others of")

	organiseFlags := flag.NewFlagSet("organise", flag.ContinueOnError)
	var doSlotsMoves = organiseFlags.Bool("slotmoves", false, "Include slot moves in org")

	wantFlags := flag.NewFlagSet("wants", flag.ExitOnError)
	var wantID = wantFlags.Int("id", 0, "Id of the want")
	var wantValue = wantFlags.Bool("want", false, "Whether to value the want")

	spendFlags := flag.NewFlagSet("spend", flag.ContinueOnError)
	var doList = spendFlags.Bool("list", false, "Show the records spent on")
	var justPrint = spendFlags.Bool("justprint", false, "Just print the spend")

	deleteWantFlags := flag.NewFlagSet("deletewant", flag.ExitOnError)
	var deleteWantID = deleteWantFlags.Int("id", 0, "Id of want to delete")

	updateDateAddedFlags := flag.NewFlagSet("setadded", flag.ExitOnError)
	var udaID = updateDateAddedFlags.Int("id", 0, "Id of record to update")
	var udaDate = updateDateAddedFlags.String("date", "", "The date to update to.")

	sellFlags := flag.NewFlagSet("sell", flag.ExitOnError)
	var sellID = sellFlags.Int("id", 0, "Id of record to sell")

	printTidyFlags := flag.NewFlagSet("printtidy", flag.ExitOnError)
	var ptLoc = printTidyFlags.String("name", "", "The folder to print")

	deleteFlags := flag.NewFlagSet("delete", flag.ExitOnError)
	var instance = deleteFlags.Int("instance", 0, "Instance to delete")

	rawLocFlags := flag.NewFlagSet("rawlocation", flag.ExitOnError)
	var rawID = rawLocFlags.Int("id", 0, "Folder to investigate")

	delLocFlags := flag.NewFlagSet("deleteLocation", flag.ExitOnError)
	var delLocName = delLocFlags.String("name", "", "The name of the folder to delete")

	var quiet = flag.Bool("quiet", false, "Show all output")
	flag.Parse()

	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	switch os.Args[1] {
	case "add":
		if err := addFlags.Parse(os.Args[2:]); err == nil {
			addRecord(*id)
		}
	case "addlocation":
		if err := addLocationFlags.Parse(os.Args[2:]); err == nil {
			addLocation(*name, *units, *folderIds)
		}
	case "getLocation":
		if err := getLocationFlags.Parse(os.Args[2:]); err == nil {
			getLocation(*getName, int32(*slot), *timestamp)
		}
	case "listFolders":
		listFolders()
	case "move":
		if err := moveFlags.Parse(os.Args[2:]); err == nil && *idToMoveToFolder > 0 {
			move(*idToMoveToFolder, *folderID)
		}
	case "organise":
		if err := organiseFlags.Parse(os.Args[2:]); err == nil {
			organise(*doSlotsMoves)
		}
	case "locate":
		if err := locateFlags.Parse(os.Args[2:]); err == nil && *idToLocate > 0 {
			locate(*idToLocate)
		}
	case "updatelocation":
		if err := updateLocationFlags.Parse(os.Args[2:]); err == nil {
			location := &pbo.Location{Name: *nameToUpdate}
			if len(*sort) > 0 {
				switch *sort {
				case "by_label":
					location.Sort = pbo.Location_BY_LABEL_CATNO
				case "by_date":
					location.Sort = pbo.Location_BY_DATE_ADDED
				case "by_release":
					location.Sort = pbo.Location_BY_RELEASE_DATE
				}
			} else if len(*updateFolders) > 0 {
				location.FolderIds = make([]int32, 0)
				for _, folder := range strings.Split(*updateFolders, ",") {
					folderID, _ := strconv.Atoi(folder)
					location.FolderIds = append(location.FolderIds, int32(folderID))
				}
			} else if *numSlots > 0 {
				location.Units = int32(*numSlots)
			} else if *formatexp != "" {
				location.ExpectedFormat = *formatexp
			} else if *unexpectedlabel != "" {
				location.UnexpectedLabel = *unexpectedlabel
			} else if *quotaNum > 0 {
				location.Quota = &pbo.Quota{NumOfUnits: int32(*quotaNum)}
			}
			updateLocation(location)
		}
	case "investigate":
		if err := investigateFlags.Parse(os.Args[2:]); err == nil {
			if *investigateYear > 0 {
				for _, rel := range getAllReleases() {
					_, meta := getRelease(rel.Id)
					if time.Unix(meta.DateAdded, 0).Year() == int(*investigateYear) {
						fmt.Printf("%v\n", prettyPrintRelease(rel.Id))
					}
				}
			} else {
				if *deepInvestigate {
					recs := getAllReleases()

					for _, r := range recs {
						if int(r.Id) == *investigateID {
							fmt.Printf("%v\n", r)
						}
					}
				} else {
					rel, meta := getRelease(int32(*investigateID))
					fmt.Printf("%v\n%v\n", rel, meta)
				}
			}
		}
	case "diff":
		if err := diffFlags.Parse(os.Args[2:]); err == nil {
			differ := &pbo.DiffRequest{
				Slot:         int32(*diffSlot),
				LocationName: *diffName,
			}
			printDiff(differ)
		}
	case "low":
		if err := lowFlags.Parse(os.Args[2:]); err == nil {
			printLow(*lowFolderName, *showOthersOnly)
		}
	case "wantlist":
		printWantlist()
	case "collapse":
		collapseWantlist()
	case "rebuild":
		rebuildWantlist()
	case "printspend":
		spend, records := getSpend()
		n := time.Now()
		//Allowed spend is $400 dollars over two months
		allowedSpendPerQuarter := 40000.0 * 3.0 * float64(n.YearDay()%91) / 91.0
		if err := spendFlags.Parse(os.Args[2:]); err == nil {
			fmt.Printf("Spend = %v / %v [%v]\n", spend, allowedSpendPerQuarter, *doList)
			if *justPrint {
				return
			}
			if *doList {
				for i, record := range records {
					fmt.Printf("%v. %v [%v]\n", i, prettyPrintRelease(record.Release.Id), record.Update.Cost)
				}
			} else {
				if float64(spend) > allowedSpendPerQuarter {
					fmt.Printf("Collapsing Wantlist")
					collapseWantlist()
				} else {
					fmt.Printf("Restoring Wantlist")
					rebuildWantlist()
				}
				log.Printf("WANTLIST")
			}
		}
	case "want":
		if err := wantFlags.Parse(os.Args[2:]); err == nil {
			setWant(*wantID, *wantValue)
		}
	case "deletewant":
		if err := deleteWantFlags.Parse(os.Args[2:]); err == nil {
			if *deleteWantID < 0 {
				deleteAllWants()
			} else {
				deleteWant(*deleteWantID)
			}
		}
	case "updatemeta":
		if err := updateDateAddedFlags.Parse(os.Args[2:]); err == nil {
			updateMeta(*udaID, *udaDate)
		}
	case "sell":
		if err := sellFlags.Parse(os.Args[2:]); err == nil {
			sell(*sellID)
		}
	case "printtidy":
		if err := printTidyFlags.Parse(os.Args[2:]); err == nil {
			printTidy(*ptLoc)
		}
	case "delete":
		if err := deleteFlags.Parse(os.Args[2:]); err == nil {
			delete(*instance)
		}
	case "sync":
		dServer, dPort := getIP("discogssyncer")
		dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		defer dConn.Close()
		dClient := pb.NewDiscogsServiceClient(dConn)
		_, err = dClient.SyncWithDiscogs(context.Background(), &pb.Empty{})
		if err != nil {
			log.Fatalf("Failure to sync: %v", err)
		}
	case "rawlocation":
		if err := rawLocFlags.Parse(os.Args[2:]); err == nil {
			listFolder(int32(*rawID))
		}
	case "deletelocation":
		if err := delLocFlags.Parse(os.Args[2:]); err == nil {
			deleteLocation(*delLocName)
		}
	}

}
