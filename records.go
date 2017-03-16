package main

import (
	"flag"
	"fmt"
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
	pbo "github.com/brotherlogic/recordsorganiser/proto"
)

func getIP(servername string, ip string, port int) (string, int) {
	conn, _ := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	defer conn.Close()

	registry := pbdi.NewDiscoveryServiceClient(conn)
	entry := pbdi.RegistryEntry{Name: servername}
	r, _ := registry.Discover(context.Background(), &entry)
	return r.Ip, int(r.Port)
}

func addLocation(name string, units int, folders string) {
	location := &pbo.Location{Name: name, Units: int32(units)}
	for _, folder := range strings.Split(folders, ",") {
		folderID, _ := strconv.Atoi(folder)
		location.FolderIds = append(location.FolderIds, int32(folderID))
	}

	//Move the previous record down to uncategorized
	dServer, dPort := getIP("recordsorganiser", "192.168.86.34", 50055)
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer dConn.Close()
	dClient := pbo.NewOrganiserServiceClient(dConn)
	log.Printf("Sending: %v", location)
	newLocation, _ := dClient.AddLocation(context.Background(), location)
	log.Printf("New Location = %v", newLocation)
}

func addRecord(id int) {
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)

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
	server, port := getIP("recordsorganiser", "192.168.86.34", 50055)
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	locationQuery := &pbo.Location{Name: name, Timestamp: timestamp}
	location, err := client.GetLocation(context.Background(), locationQuery)

	fmt.Printf("%v sorted %v\n", location.Name, location.Sort)

	if err != nil {
		panic(err)
	}
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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

		for _, rel := range releases.Releases {
			relMap[rel.Id] = rel
		}
	}

	for _, release := range location.ReleasesLocation {
		if release.Slot == slot {
			fullRelease, err := dClient.GetSingleRelease(context.Background(), &pbd.Release{Id: release.ReleaseId})
			if err == nil {
				fmt.Printf("%v. %v - %v\n", release.Index, pbd.GetReleaseArtist(*fullRelease), fullRelease.Title)
			}
		}
	}

}

func getRelease(id int32) (*pbd.Release, *pb.ReleaseMetadata) {
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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
		return pbd.GetReleaseArtist(*rel) + " - " + rel.Title
	}
	if id == 0 {
		return "---------------"
	}
	return strconv.Itoa(int(id))
}

func listUncategorized() {
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)
	releases, err := dClient.GetReleasesInFolder(context.Background(), &pb.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: 1}}})

	if err != nil {
		panic(err)
	}

	for _, release := range releases.Releases {
		fmt.Printf("%v: %v - %v\n", release.Id, pbd.GetReleaseArtist(*release), release.Title)
	}
}
func listFolders() {
	server, port := getIP("recordsorganiser", "192.168.86.34", 50055)
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
	server, port := getIP("recordsorganiser", "192.168.86.34", 50055)
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
	server, port := getIP("recordsorganiser", "192.168.86.34", 50055)
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	log.Printf("Updating: %v", loc)
	client.UpdateLocation(context.Background(), loc)
}

func listCollections() {
	server, port := getIP("recordsorganiser", "192.168.86.34", 50055)
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	orgs, err := client.GetOrganisations(context.Background(), &pbo.Empty{})

	if err != nil {
		panic(err)
	}

	if len(orgs.Organisations) == 0 {
		fmt.Printf("There are no stored orgs\n")
	}

	for _, org := range orgs.Organisations {
		fmt.Printf("%v\n", org.Timestamp)
	}
}

func printDiff(diffRequest *pbo.DiffRequest) {
	server, port := getIP("recordsorganiser", "192.168.86.34", 50055)
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
	server, port := getIP("recordsorganiser", "192.168.86.34", 50055)
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	res, _ := client.Locate(context.Background(), &pbd.Release{Id: int32(id)})

	fmt.Printf("In %v, slot %v\n", res.Location.Name, res.Slot)
	if res.Before != nil {
		fmt.Printf("Before: %v - %v (%v)\n", pbd.GetReleaseArtist(*res.Before), res.Before.Title, res.Before.Id)
	}
	if res.After != nil {
		fmt.Printf("After:  %v - %v (%v)\n", pbd.GetReleaseArtist(*res.After), res.After.Title, res.After.Id)
	}
}

func moveToPile(id int) {
	log.Printf("Moving")
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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

	for _, release := range releases.Releases {
		if release.Id == int32(id) {
			move := &pb.ReleaseMove{Release: &pbd.Release{Id: int32(id), FolderId: 1, InstanceId: release.InstanceId}, NewFolderId: int32(812802)}
			_, err = dClient.MoveToFolder(context.Background(), move)
			log.Printf("MOVED %v from %v", move, release)
			if err != nil {
				panic(err)
			}
		}
	}
}

func collapseWantlist() {
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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

func printWantlist() {
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	//Two month rolling average
	now := time.Now()
	before := now.AddDate(0, -2, 0)
	spend, err := dClient.GetSpend(context.Background(), &pb.SpendRequest{Lower: before.Unix(), Upper: now.Unix()})
	if err != nil {
		panic(err)
	}
	return int(spend.TotalSpend), spend.GetSpends()
}

func setWant(id int, wantValue bool) {
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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

func printLow(name string, others bool) {
	//Move the previous record down to uncategorized
	server, port := getIP("recordsorganiser", "192.168.86.34", 50055)
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

	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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
		releases, err := dClient.GetReleasesInFolder(context.Background(), &pb.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: folderID}}})

		if err != nil {
			panic(err)
		}

		for _, release := range releases.Releases {
			if int(release.Rating) < lowestScore {
				lowestScore = int(release.Rating)
				lowest = make([]*pbd.Release, 0)
				lowest = append(lowest, release)
			} else if int(release.Rating) == lowestScore {
				lowest = append(lowest, release)
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
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)

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
}

func printTidy(place string) {
	//Move the previous record down to uncategorized
	server, port := getIP("recordsorganiser", "192.168.86.34", 50055)
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
	dServer, dPort := getIP("discogssyncer", "192.168.86.34", 50055)
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

		for _, rel := range releases.Releases {
			relMap[rel.Id] = rel
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
			fmt.Printf("%v. %v - %v\n", slotv, pbd.GetReleaseArtist(*bestRelease[slotv]), bestRelease[slotv].Title)
			slotv++
		} else {
			slotv = -1
		}
	}
}

func main() {
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

	moveToPileFlags := flag.NewFlagSet("MoveToPile", flag.ContinueOnError)
	var idToMove = moveToPileFlags.Int("id", 0, "Id of record to move")

	locateFlags := flag.NewFlagSet("Locate", flag.ExitOnError)
	var idToLocate = locateFlags.Int("id", 0, "Id of record to locate")

	updateLocationFlags := flag.NewFlagSet("UpdateLocation", flag.ContinueOnError)
	var nameToUpdate = updateLocationFlags.String("name", "", "Name of the location to update")
	var sort = updateLocationFlags.String("sort", "", "Sorting method of the location")
	var updateFolders = updateLocationFlags.String("folders", "", "Folders to add")
	var numSlots = updateLocationFlags.Int("slots", -1, "The number of slots to update to")

	investigateFlags := flag.NewFlagSet("investigate", flag.ExitOnError)
	var investigateID = investigateFlags.Int("id", 0, "Id of release to investigate")
	var investigateYear = investigateFlags.Int("year", 0, "Year of releases to list")

	diffFlags := flag.NewFlagSet("diff", flag.ExitOnError)
	var startTimestamp = diffFlags.Int64("start", 0, "Start timestamp")
	var endTimestamp = diffFlags.Int64("end", 0, "End timestamp")
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

	deleteWantFlags := flag.NewFlagSet("deletewant", flag.ExitOnError)
	var deleteWantID = deleteWantFlags.Int("id", 0, "Id of want to delete")

	updateDateAddedFlags := flag.NewFlagSet("setadded", flag.ExitOnError)
	var udaID = updateDateAddedFlags.Int("id", 0, "Id of record to update")
	var udaDate = updateDateAddedFlags.String("date", "", "The date to update to.")

	sellFlags := flag.NewFlagSet("sell", flag.ExitOnError)
	var sellID = sellFlags.Int("id", 0, "Id of record to sell")

	printTidyFlags := flag.NewFlagSet("printtidy", flag.ExitOnError)
	var ptLoc = printTidyFlags.String("name", "", "The folder to print")

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
	case "listTimes":
		listCollections()
	case "listFolders":
		listFolders()
	case "uncat":
		if err := moveToPileFlags.Parse(os.Args[2:]); err == nil && *idToMove > 0 {
			moveToPile(*idToMove)
			listUncategorized()
		} else {
			listUncategorized()
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
				rel, meta := getRelease(int32(*investigateID))
				fmt.Printf("%v\n%v\n", rel, meta)
			}
		}
	case "diff":
		if err := diffFlags.Parse(os.Args[2:]); err == nil {
			differ := &pbo.DiffRequest{
				StartTimestamp: *startTimestamp,
				EndTimestamp:   *endTimestamp,
				Slot:           int32(*diffSlot),
				LocationName:   *diffName,
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
		//Allowed spend is $400 dollars over two months
		allowedSpend := 40000.0 * 2.0
		if err := spendFlags.Parse(os.Args[2:]); err == nil {
			fmt.Printf("Spend = %v / %v [%v]\n", spend, allowedSpend, *doList)
			if *doList {
				for i, record := range records {
					fmt.Printf("%v. %v [%v]\n", i, prettyPrintRelease(record.Release.Id), record.Update.Cost)
				}
			} else {
				if float64(spend) > allowedSpend {
					fmt.Printf("Collapsing Wantlist")
					collapseWantlist()
				} else {
					fmt.Printf("Restoring Wantlist")
					rebuildWantlist()
				}
			}
		}
	case "want":
		if err := wantFlags.Parse(os.Args[2:]); err == nil {
			setWant(*wantID, *wantValue)
		}
	case "deletewant":
		if err := deleteWantFlags.Parse(os.Args[2:]); err == nil {
			deleteWant(*deleteWantID)
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
	}

}
