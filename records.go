package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/brotherlogic/discogssyncer/server"
	pbd "github.com/brotherlogic/godiscogs"
	"github.com/brotherlogic/goserver/utils"
)

func getIP(name string) (string, int) {
	ip, port, _ := utils.Resolve(name)
	return ip, int(port)
}

func listFolder(ID int32) {
	dServer, dPort := getIP("discogssyncer")
	//Move the previous record down to uncategorized
	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(int(dPort)), grpc.WithInsecure())
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

	moveFlags := flag.NewFlagSet("Move", flag.ExitOnError)
	var idToMoveToFolder = moveFlags.Int("id", 0, "Id of record to move")
	var folderID = moveFlags.Int("folder", 0, "Id of folder to move to")

	investigateFlags := flag.NewFlagSet("investigate", flag.ExitOnError)
	var investigateID = investigateFlags.Int("id", 0, "Id of release to investigate")
	var investigateYear = investigateFlags.Int("year", 0, "Year of releases to list")
	var deepInvestigate = investigateFlags.Bool("deep", false, "Do a deep search for the release")

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

	deleteFlags := flag.NewFlagSet("delete", flag.ExitOnError)
	var instance = deleteFlags.Int("instance", 0, "Instance to delete")

	rawLocFlags := flag.NewFlagSet("rawlocation", flag.ExitOnError)
	var rawID = rawLocFlags.Int("id", 0, "Folder to investigate")

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
	case "move":
		if err := moveFlags.Parse(os.Args[2:]); err == nil && *idToMoveToFolder > 0 {
			move(*idToMoveToFolder, *folderID)
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
	}
}
