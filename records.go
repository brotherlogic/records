package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

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
	dServer, dPort := getIP("recordsorganiser", "10.0.1.17", 50055)
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
	dServer, dPort := getIP("discogssyncer", "10.0.1.17", 50055)

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

func getLocation(name string, slot int32) {
	//Move the previous record down to uncategorized
	server, port := getIP("recordsorganiser", "10.0.1.17", 50055)
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	locationQuery := &pbo.Location{Name: name}
	location, err := client.GetLocation(context.Background(), locationQuery)

	fmt.Printf("%v sorted %v\n", location.Name, location.Sort)

	if err != nil {
		panic(err)
	}
	dServer, dPort := getIP("discogssyncer", "10.0.1.17", 50055)
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
			fmt.Printf("%v. %v - %v\n", release.Index, pbd.GetReleaseArtist(*relMap[release.ReleaseId]), relMap[release.ReleaseId].Title)
		}
	}

}

func listUncategorized() {
	dServer, dPort := getIP("discogssyncer", "10.0.1.17", 50055)
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
	server, port := getIP("recordsorganiser", "10.0.1.17", 50055)
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

func organise() {
	server, port := getIP("recordsorganiser", "10.0.1.17", 50055)
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

	if len(moves.Moves) == 0 {
		fmt.Printf("No Moves needed\n")
	}

	for _, move := range moves.Moves {
		fmt.Printf("MOVE %v\n", move)
	}
}

func updateLocation(loc *pbo.Location) {
	server, port := getIP("recordsorganiser", "10.0.1.17", 50055)
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
	server, port := getIP("recordsorganiser", "10.0.1.17", 50055)
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

func locate(id int) {
	server, port := getIP("recordsorganiser", "10.0.1.17", 50055)
	conn, err := grpc.Dial(server+":"+strconv.Itoa(port), grpc.WithInsecure())

	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pbo.NewOrganiserServiceClient(conn)
	res, _ := client.Locate(context.Background(), &pbd.Release{Id: int32(id)})

	fmt.Printf("In %v, slot %v\n", res.Location.Name, res.Slot)
	fmt.Printf("Before: %v - %v (%v)\n", pbd.GetReleaseArtist(*res.Before), res.Before.Title, res.Before.Id)
	fmt.Printf("After:  %v - %v (%v)\n", pbd.GetReleaseArtist(*res.After), res.After.Title, res.After.Id)
}

func moveToPile(id int) {
	log.Printf("Moving")
	dServer, dPort := getIP("discogssyncer", "10.0.1.17", 50055)
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(dPort), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer dConn.Close()
	dClient := pb.NewDiscogsServiceClient(dConn)

	releases, err := dClient.GetReleasesInFolder(context.Background(), &pb.FolderList{Folders: []*pbd.Folder{&pbd.Folder{Id: 1}}})

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

func main() {
	addFlags := flag.NewFlagSet("AddRecord", flag.ExitOnError)
	var id = addFlags.Int("id", 0, "ID of record to add")

	addLocationFlags := flag.NewFlagSet("AddLocation", flag.ExitOnError)
	var name = addLocationFlags.String("name", "", "The name of the new location")
	var units = addLocationFlags.Int("units", 0, "The number of units in the location")
	var folderIds = addLocationFlags.String("folders", "", "The list of folder IDs")

	getLocationFlags := flag.NewFlagSet("GetLocation", flag.ExitOnError)
	var getName = getLocationFlags.String("name", "", "The name of the location to get")
	var slot = getLocationFlags.Int("slot", 0, "The slot to retrieve from")

	moveToPileFlags := flag.NewFlagSet("MoveToPile", flag.ContinueOnError)
	var idToMove = moveToPileFlags.Int("id", 0, "Id of record to move")

	locateFlags := flag.NewFlagSet("Locate", flag.ExitOnError)
	var idToLocate = locateFlags.Int("id", 0, "Id of record to locate")

	updateLocationFlags := flag.NewFlagSet("UpdateLocation", flag.ContinueOnError)
	var nameToUpdate = updateLocationFlags.String("name", "", "Name of the location to update")
	var sort = updateLocationFlags.String("sort", "", "Sorting method of the location")

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
			getLocation(*getName, int32(*slot))
		}
	case "listTimes":
		listCollections()
	case "listFolders":
		listFolders()
	case "uncat":
		if err := moveToPileFlags.Parse(os.Args[2:]); err == nil && *idToMove > 0 {
			moveToPile(*idToMove)
		} else {
			listUncategorized()
		}
		listUncategorized()
	case "organise":
		organise()
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
				}
			}
			log.Printf("HERE: %v", location)
			updateLocation(location)
		}
	}
}
