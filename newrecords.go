package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/brotherlogic/goserver/utils"
	"google.golang.org/grpc"

	pbrc "github.com/brotherlogic/recordcollection/proto"
)

func main() {
	dServer, dPort, err := utils.Resolve("recordcollection")
	if err != nil {
		log.Fatalf("Error in resolving recordcollection: %v", err)
	}

	dConn, err := grpc.Dial(dServer+":"+strconv.Itoa(int(dPort)), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error dialling recordcollection")
	}
	defer dConn.Close()
	client := pbrc.NewRecordCollectionServiceClient(dConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Argument handler
	switch os.Args[1] {
	case "uncat":
		err := listUncategorized(ctx, client)
		if err != nil {
			fmt.Printf("Error in list uncategorized: %v", err)
		}
	case "add":
		addRecordFlags := flag.NewFlagSet("addrecord", flag.ExitOnError)
		var id = addRecordFlags.Int("id", 0, "The id of the record")
		var cost = addRecordFlags.Int("cost", 0, "The cost of the record (in cents)")
		var folder = addRecordFlags.Int("folder", 0, "The id of the folder that this'll end up in")

		if err := addRecordFlags.Parse(os.Args[2:]); err == nil {
			_, err := add(ctx, client, int32(*id), int32(*cost), int32(*folder))
			if err != nil {
				log.Fatalf("Error adding record: %v", err)
			}
		}
	}
}
