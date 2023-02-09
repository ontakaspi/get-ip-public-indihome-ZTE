package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func cronJob() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	s := gocron.NewScheduler(loc)
	//downloadTrivyDB() every day at 12:00 AM
	go func() {
		s.Every(10).Minute().Do(func() {
			IPPublic := GetIPPublic()
			if IPPublic != "" {
				updateDNSRecordCloudFare(IPPublic)
			}
		})
	}()

	s.StartAsync()
}

func main() {
	// Start the cron job
	cronJob()

	// to run the function without cron (HTTP API)
	http.HandleFunc("/do-refresh", func(w http.ResponseWriter, r *http.Request) {
		// Start a Selenium WebDriver server instance
		IPPublic := GetIPPublic()
		if IPPublic != "" {
			updateDNSRecordCloudFare(IPPublic)
		}
	})

	http.ListenAndServe(":8080", nil)
	fmt.Println("Server started on port 8080")

	// Start a Selenium WebDriver server instance
	IPPublic := GetIPPublic()
	updateDNSRecordCloudFare(IPPublic)

}

func GetIPPublic() string {
	selenium.SetDebug(false)
	// Start a Selenium WebDriver with headless Chrome
	fmt.Println("Starting Selenium WebDriver")
	service, err := selenium.NewChromeDriverService("/usr/bin/chromedriver", 4444)
	if err != nil {
		fmt.Println("Error starting Selenium WebDriver:", err)
		return ""
	}
	defer service.Stop()

	// Connect to the WebDriver instance running locally and start a new browser session
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}
	chromeCaps := chrome.Capabilities{
		Path: "",
		Args: []string{
			"--headless", // <<<
			"--no-sandbox",
			"--user-agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_2) AppleWebKit/604.4.7 (KHTML, like Gecko) Version/11.0.2 Safari/604.4.7",
		},
	}
	caps.AddChrome(chromeCaps)

	fmt.Println("Connecting to the WebDriver instance running locally and start a new browser session")
	wd, err := selenium.NewRemote(caps, "")
	if err != nil {
		fmt.Println("Error starting Selenium WebDriver:", err)
		return ""
	}
	defer wd.Quit()
	if err != nil {
		fmt.Println("Error starting Selenium WebDriver:", err)
		return ""
	}

	err = loginToRouter(err, wd)
	if err != nil {
		fmt.Println("Error logging in to the router:", err)
		return ""
	}

	wanIp, err := getWanIPAddress(wd)
	if err != nil {
		fmt.Println("Error getting WAN IP Address:", err)
		return ""
	}
	//check if the WAN IP Address is a public IP Address not 10.x.x.x
	var wanIpSplit []string
	wanIpSplit = strings.Split(wanIp, ".")
	for wanIpSplit[0] == "10" {
		fmt.Println("CURRENT WAN IP Address:", wanIp)
		fmt.Println("WAN IP Address is not a public IP Address, refreshing the WAN IP Address")
		//refresh the WAN IP Address
		err = refreshTheIpAddress(wd)
		if err != nil {
			fmt.Println("Error refreshing the WAN IP Address:", err)
			return ""
		}

		//get the WAN IP Address
		wanIp, err = getWanIPAddress(wd)
		if err != nil {
			fmt.Println("Error getting WAN IP Address:", err)
			return ""
		}
		wanIpSplit = strings.Split(wanIp, ".")
	}

	fmt.Println("WAN IP Address is a public IP Address:", wanIp)

	var IpSplit []string
	IpSplit = strings.Split(wanIp, "/")
	return IpSplit[0]
}
func loginToRouter(err error, wd selenium.WebDriver) error {
	// Navigate to the login page
	err = wd.Get("http://192.168.1.1")
	if err != nil {
		return err
	}
	// Find the login form elements
	username, err := wd.FindElement(selenium.ByID, "Frm_Username")
	if err != nil {
		return err
	}
	password, err := wd.FindElement(selenium.ByID, "Frm_Password")
	if err != nil {
		return err
	}
	submit, err := wd.FindElement(selenium.ByID, "LoginId")
	if err != nil {
		return err
	}

	// Enter the login credentials and submit the form
	err = username.SendKeys("admin")
	if err != nil {
		return err
	}
	err = password.SendKeys("xxxxx")
	if err != nil {
		return err
	}
	err = submit.Click()
	if err != nil {
		return err
	}
	return err
}

func getWanIPAddress(wd selenium.WebDriver) (string, error) {
	ipAdd, err := logicGetWanIPAddress(wd)
	if err != nil {
		return "", err
	}
	for strings.Contains(ipAdd, "0.0.0.0") {
		fmt.Println("Waiting for WAN IP Address to be assigned")
		time.Sleep(time.Duration(2) * time.Second)
		ipAdd, err = logicGetWanIPAddress(wd)
	}

	return ipAdd, nil
}

func logicGetWanIPAddress(wd selenium.WebDriver) (string, error) {
	sleepTime := 2
	sleepTime = sleepTime * 1000
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	// Click the Tab Internet and wait for the page to load
	tabInternet, err := wd.FindElement(selenium.ByID, "internet")
	if err != nil {
		return "", err
	}

	err = tabInternet.Click()
	if err != nil {
		return "", err
	}

	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	// Click the Status menu and wait for the page to load
	status, err := wd.FindElement(selenium.ByID, "internetStatus")
	if err != nil {
		return "", err
	}
	err = status.Click()
	if err != nil {
		return "", err
	}

	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	// Click the tab WAN and wait for the page to load
	tabWAN, err := wd.FindElement(selenium.ByID, "ethWanStatus")
	if err != nil {
		return "", err
	}
	err = tabWAN.Click()
	if err != nil {
		return "", err
	}

	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	// get the WAN IP Address
	wanIP, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[3]/div[2]/div[1]/div[3]/div[2]/div[2]/div/div/div[2]/div[3]/form/div[5]/span[2]")
	if err != nil {
		return "", err
	}
	wanIPText, err := wanIP.Text()
	if err != nil {
		return "", err
	}

	return wanIPText, nil
}

func refreshTheIpAddress(wd selenium.WebDriver) error {
	//wait for the selector to be clickable
	sleepTime := 2

	sleepTime = sleepTime * 1000
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	err := wd.Wait(func(wd selenium.WebDriver) (bool, error) {
		refresh, err := wd.FindElement(selenium.ByID, "internet")
		if err != nil {
			return false, err
		}
		return refresh.IsDisplayed()
	})
	// Click the Tab Internet and wait for the page to load
	tabInternet, err := wd.FindElement(selenium.ByID, "internet")
	if err != nil {
		return err
	}

	err = tabInternet.Click()
	if err != nil {
		return err
	}

	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	// Click the Status menu and wait for the page to load
	status, err := wd.FindElement(selenium.ByID, "internetConfig")
	if err != nil {
		return err
	}
	err = status.Click()
	if err != nil {
		return err
	}

	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	// Click the tag span with attribute title="INTERNET"
	tabInternet, err = wd.FindElement(selenium.ByCSSSelector, "span[title='INTERNET']")
	if err != nil {
		return err
	}
	err = tabInternet.Click()
	if err != nil {
		return err
	}

	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	var ChangeValue string
	// Get The input select near label "Authentication Type"
	AuthTypeAuto := isSelected(wd, "Auto")
	if !AuthTypeAuto {
		AuthTypePAAP := isSelected(wd, "PAP")
		if !AuthTypePAAP {
			return errors.New("authentication Type is not Auto or PAP")
		}
		ChangeValue = "Auto"
	} else {
		ChangeValue = "PAP"
	}

	// Click the input select near label "Authentication Type"
	selectData, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[3]/div[2]/div[1]/div[3]/div[2]/div[2]/div/div/div[2]/div[2]/div[2]/form/div[8]/div[4]/div/select")
	if err != nil {
		return err
	}
	err = selectData.Click()

	if err != nil {
		return err
	}

	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	// Click the option near label "Authentication Type"
	if ChangeValue == "Auto" {
		selectData, err = wd.FindElement(selenium.ByXPATH, "/html/body/div[3]/div[2]/div[1]/div[3]/div[2]/div[2]/div/div/div[2]/div[2]/div[2]/form/div[8]/div[4]/div/select/option[1]")
	} else {
		selectData, err = wd.FindElement(selenium.ByXPATH, "/html/body/div[3]/div[2]/div[1]/div[3]/div[2]/div[2]/div/div/div[2]/div[2]/div[2]/form/div[8]/div[4]/div/select/option[2]")
	}

	//submit the form
	submit, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[3]/div[2]/div[1]/div[3]/div[2]/div[2]/div/div/div[2]/div[2]/div[2]/form/div[14]/input[2]")
	if err != nil {
		return err
	}
	err = submit.Click()
	return nil
}

func isSelected(wd selenium.WebDriver, typeAuth string) bool {

	var index int
	if typeAuth == "Auto" {
		index = 1
	} else {
		index = 2
	}
	selectData, err := wd.FindElement(selenium.ByXPATH, "/html/body/div[3]/div[2]/div[1]/div[3]/div[2]/div[2]/div/div/div[2]/div[2]/div[2]/form/div[8]/div[4]/div/select/option["+strconv.Itoa(index)+"]")

	if err != nil {
		return false
	}

	//get the selected value
	AuthType, err := selectData.IsSelected()
	if err != nil {
		return false
	}
	return AuthType
}

func updateDNSRecordCloudFare(IP string) {
	apiKey := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	zoneID := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	// retrieve a list of DNS records for the zone
	dnsRecords, err := getDNSRecords(apiKey, zoneID)
	if err != nil {
		fmt.Println("Error retrieving DNS records:", err)
		return
	}
	//get the DNS record where the name is "lab.kasfi-dev.tech"
	var dnsRecord DNSRecord
	for _, record := range dnsRecords {
		if record.Name == "example.hostname.com" {
			dnsRecord = record
		}
	}

	//check if the WAN IP Address is the same as the DNS record

	if IP == dnsRecord.Content {
		fmt.Println("WAN IP Address is the same as the DNS record (", IP, "), no need to update the DNS record")
		return
	}

	fmt.Println("WAN IP Address is not the same as the DNS record, updating the DNS record")
	//update the DNS record
	dnsRecord.Content = IP
	err = updateDNSRecord(apiKey, zoneID, dnsRecord)
	if err != nil {
		fmt.Println("Error updating DNS record:", err)
		return
	}

	fmt.Println("DNS record updated successfully")
}

type DNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
}

func getDNSRecords(apiKey, zoneID string) ([]DNSRecord, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Key", apiKey)
	req.Header.Set("X-Auth-Email", "youremail@gmail.com")
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result []DNSRecord `json:"result"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result.Result, nil
}

func updateDNSRecord(apiKey, zoneID string, dnsRecord DNSRecord) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, dnsRecord.ID)

	data, err := json.Marshal(dnsRecord)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("X-Auth-Key", apiKey)
	req.Header.Set("X-Auth-Email", "youremail@gmail.com")
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Result DNSRecord `json:"result"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}
	fmt.Println("DNS Record updated:", result.Result)
	return nil
}
