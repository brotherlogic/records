package main

import (
	"context"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Argument handler
	switch os.Args[1] {
	case "uncat":
		err := listUncategorized(ctx, client)
		if err != nil {
			fmt.Printf("Error in list uncategorized: %v", err)
		}
	}
}
