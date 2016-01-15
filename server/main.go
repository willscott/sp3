package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	configFile *string = flag.String("config", "", "File with server configuration")
	initFlag   *bool   = flag.Bool("init", false, "if true, setup new configuration")
	port       *int    = flag.Int("port", 8080, "TCP port for connections")
	device     *string = flag.String("device", "eth0", "inet device for pcap to use")
	srcMAC     *string = flag.String("srcMAC", "000000000000", "Ethernet SRC for sending")
	dstMAC     *string = flag.String("dstMAC", "000000000000", "Ethernet DST for sending")
)

type Config struct {
	port int
	device string
	src string
	dst string
}

func main() {
	flag.Parse()

	if len(*configFile) == 0 {
		home := os.Getenv("HOME")
		if len(home) == 0 {
			fmt.Fprintf(os.Stderr, "$HOME not set. Please either export $HOME or use an explict --config location.\n")
			os.Exit(1)
		}
		configDir := filepath.Join(home, ".config")
		if *initFlag {
			os.Mkdir(configDir, 0700)
		}
		*configFile = filepath.Join(configDir, "sp3.json")
	}
	if *initFlag {
		configHandle, err := os.OpenFile(*configFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			log.Fatalf("Failed to create config file: %s", err)
			return
		}
		defaultConfig, _ := json.Marshal(Config{
			port: *port,
			device: *device,
			src: *srcMAC,
			dst: *dstMAC,
		})
		if _, err := configHandle.Write(defaultConfig); err != nil {
			log.Fatalf("Failed to write default config: %s", err)
			return
		}
		configHandle.Close()
	}

	configString, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Couldn't read config file: %s", err)
		return
	}
	config := Config{}
	if err := json.Unmarshal(configString, &config); err != nil {
		log.Fatalf("Couldn't parse config: %s", err)
		return
	}

	if config.port == 0 {
		config.port = 8080
	}
	if config.device == "" {
		config.device = "eth0"
	}

	fmt.Println("running on port", config.port)
	NewServer(config)
}
