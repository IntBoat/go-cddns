package main

import (
	"context"
	"errors"
	"github.com/cloudflare/cloudflare-go"
	"github.com/magiconair/properties"
	"github.com/spf13/viper"
	"go-cddns/version"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var logger *log.Logger
var LastIPAddress = ""
var API *cloudflare.API
var ZoneID *cloudflare.ResourceContainer
var ctx = context.Background()

func init() {
	logger = log.New(io.Writer(os.Stdout), "go-cddns:", log.Lshortfile)
}

func main() {
	logger.Println("Version:", version.BuildVersion())
	// Parse command line flags
	parseCommandLineFlags()

	//	Read config file
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath("/etc/go-cddns/")
	viper.AddConfigPath("$HOME/.go-cddns")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatalf("Cannot open config file.")
	}
	viper.WatchConfig()

	err = validateConfig()
	if err != nil {
		logger.Fatalln(err)
	}

	// Register interrupt handler
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go handleInterrupt(sigs)

	api, err := cloudflare.NewWithAPIToken(viper.GetString("Token"))
	if err != nil {
		logger.Println(err)
		return
	}
	API = api

	// Fetch the zone ID
	id, err := API.ZoneIDByName(viper.GetString("DomainName")) // Assuming example.com exists in your Cloudflare account already
	if err != nil {
		logger.Println(err)
		return
	}
	ZoneID = cloudflare.ZoneIdentifier(id)

	updateCloudflareRecord()
	for range time.NewTicker(viper.GetDuration("UpdateInterval") * time.Minute).C {
		updateCloudflareRecord()
	}

	logger.Println("done")
}

func handleInterrupt(signalChannel chan os.Signal) {
	<-signalChannel
	logger.Println("Shutting down, performing cleanup operations")
	if viper.GetBool("Remove") {
		logger.Println("Removing DNS records")

		// Fetch zone details
		recs, _, err := API.ListDNSRecords(ctx, ZoneID, cloudflare.ListDNSRecordsParams{
			Name: strings.Join(viper.GetStringSlice("RecordNames"), ","),
		})
		if err != nil {
			logger.Println(err)
			return
		}
		for _, rec := range recs {
			logger.Printf("Removing %v", rec.Name)
			err := API.DeleteDNSRecord(ctx, ZoneID, rec.ID)
			if err != nil {
				logger.Println(err)
			}
		}
	}
	logger.Println("Exiting")
	os.Exit(0)
}

func updateCloudflareRecord() {

	// Get an updated record
	updatedIPAddress, err := getCurrentIPAddress()
	if err != nil {
		logger.Println(err)
		return
	}

	if updatedIPAddress != LastIPAddress {
		logger.Printf("IP Address changed from %v, to %v\n", LastIPAddress, updatedIPAddress)
		LastIPAddress = updatedIPAddress

		// Fetch zone details
		recs, _, err := API.ListDNSRecords(ctx, ZoneID, cloudflare.ListDNSRecordsParams{
			Name: strings.Join(viper.GetStringSlice("RecordNames"), ","),
		})
		if err != nil {
			logger.Println(err)
			return
		}
		// Range over the given RecordNames and update them, if possible
		for _, r := range viper.GetStringSlice("RecordNames") {
			found := false
			for _, v := range recs {
				if r == v.Name {
					found = true

					if v.Content != LastIPAddress {
						logger.Printf("Record %v has ip address %v, updating to %v\n", v.Name, v.Content, updatedIPAddress)
						err := API.UpdateDNSRecord(ctx, ZoneID, cloudflare.UpdateDNSRecordParams{
							Type:    "A",
							Name:    v.Name,
							Content: updatedIPAddress,
							ID:      v.ID,
						})
						if err != nil {
							log.Printf("Unable to update record %v: %v\n", v.Name, err)
							return
						}
						logger.Printf("Done updating %v\n", v.Name)
					} else {
						logger.Printf("Record %v has correct IP Address, skipping\n", v.Name)
					}
					break
				}
			}
			if !found {
				//	Record doesn't exist, create it
				logger.Printf("Record %v doesn't exist, creating it with IP Address: %v\n", r, updatedIPAddress)
				_, err = API.CreateDNSRecord(ctx, ZoneID, cloudflare.CreateDNSRecordParams{
					Type:    "A",
					Name:    r,
					Content: updatedIPAddress,
				})
				if err != nil {
					logger.Println(err)
				}
				continue
			}

		}
	}
}

func getCurrentIPAddress() (string, error) {
	resp, err := properties.LoadURL("https://1.1.1.1/cdn-cgi/trace")
	return resp.GetString("ip", "127.0.0.1"), err

}

func validateConfig() error {

	// Validate credentials
	if viper.GetString("Token") == "" {
		return errors.New("empty API Key, exiting")
	}

	if viper.GetString("Email") == "" {
		return errors.New("empty Email Address, exiting")
	}

	if viper.GetString("DomainName") == "" {
		return errors.New("empty Domain Name, exiting")
	}

	if len(viper.GetStringSlice("RecordNames")) == 0 {
		return errors.New("no records to update, exiting")
	}

	if viper.GetInt("UpdateInterval") <= 0 {
		return errors.New("update interval cannot be less than 1 minutes")
	}

	logger.Printf("Updating records for: %v", viper.GetStringSlice("RecordNames"))

	//	Check for correct config values
	logger.Printf("Setting update interval to %d minutes\n", viper.GetInt("UpdateInterval"))

	return nil
}
