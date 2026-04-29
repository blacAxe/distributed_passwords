package main

import (
	"fmt"
	"os"
	"time"
	"strings" // Add this

	"github.com/hashicorp/memberlist"
)

func main() {
	hostname, _ := os.Hostname()
	config := memberlist.DefaultLocalConfig()
	config.Name = fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano()%1000)

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
		fmt.Printf("Cluster Members: %d\n", list.NumMembers())
		time.Sleep(5 * time.Second)
	}
}