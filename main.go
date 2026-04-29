package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
)

func main() {
	hostname, _ := os.Hostname()
	config := memberlist.DefaultLocalConfig()
	// The timestamp here acts as our "seniority"
	config.Name = fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano()%1000000)

	// Check for a custom bind port (so we can run multiple on one machine)
	bindPort := os.Getenv("BIND_PORT")
	if bindPort != "" {
		var p int
		fmt.Sscanf(bindPort, "%d", &p)
		config.BindPort = p
	}

	list, err := memberlist.Create(config)
	if err != nil {
		panic(err)
	}

	local := list.LocalNode()
	fmt.Printf("Node [%s] is alive at %s:%d\n", local.Name, local.Addr, local.Port)

	// Look for an environment variable called JOIN_ADDR
	joinAddr := os.Getenv("JOIN_ADDR")
	if joinAddr != "" {
		// Split multiple addresses if provided (comma separated)
		parts := strings.Split(joinAddr, ",")
		_, err := list.Join(parts)
		if err != nil {
			fmt.Printf("Failed to join cluster: %v\n", err)
		}
	}

	for {
		// Get all members and sort them alphabetically by Name
		members := list.Members()
		sort.Slice(members, func(i, j int) bool {
			return members[i].Name < members[j].Name
		})

		// The first member in the sorted list is the Manager
		manager := members[0]

		fmt.Println("--- Cluster Status ---")
		if manager.Name == local.Name {
			fmt.Printf("ROLE: [ MANAGER ] | Nodes in cluster: %d\n", len(members))
		} else {
			fmt.Printf("ROLE: [ WORKER ]  | Manager is: %s\n", manager.Name)
		}

		time.Sleep(5 * time.Second)
	}
}