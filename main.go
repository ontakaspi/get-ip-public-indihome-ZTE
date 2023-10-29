package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"get-public-ip-indihome/libraries/logger"
	"get-public-ip-indihome/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	cors "github.com/itsjamie/gin-cors"
	"github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var routerURL = "http://192.168.x.x"
var routerAdmin = "xxxxxxxxxx"
var routerPassword = "xxxxxxxxxxxx"

var emailCloudFare = "xxxxxxxx@gmail.com"
var apiKeyCloudFare = "xxxxxxxxxxxx"

var dnsRecordNameCloudFare = "xxxxx.my.id"

var zoneIDCloudFare = "xxxxxxxxxxxxx"

func cronJob() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	s := gocron.NewScheduler(loc)
	//downloadTrivyDB() every day at 12:00 AM
	go func() {
		s.Every(10).Minute().Do(triggerCronRefreshIp)
	}()
	s.StartAsync()
}

func triggerCronRefreshIp() {
	IPPublic := GetIPPublic()
	if IPPublic != "" {
		updateDNSRecordCloudFare(IPPublic)
	}
}

var route *gin.Engine

func init() {
	//gin.SetMode(gin.ReleaseMode)
	// Initialize logger

	//setup main routes
	route = gin.New()
	route.Use(cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, PUT, POST, DELETE, OPTIONS",
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          50 * time.Second,
		ValidateHeaders: false,
	}))
	route.Use(middleware.ErrorHandler())
	route.Use(middleware.JSONMiddleware())
	route.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "this is app to set the public IP Address to the DNS record"})
	})

	route.GET("/do-refresh", func(c *gin.Context) {
		// Start a Selenium WebDriver server instance
		IPPublic := GetIPPublic()
		if IPPublic != "" {
			updateDNSRecordCloudFare(IPPublic)
		}

		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})
	route.GET("/health_check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	route.GET("/update-various-dns-record", func(c *gin.Context) {
		//get from the query parameter
		query := c.Request.URL.Query()
		//get the dns record name
		dnsRecordName := query.Get("dnsRecordName")
		if dnsRecordName != "" {
			dnsRecordNameCloudFare = dnsRecordName
		}
		//get the zone ID
		zoneID := query.Get("zoneID")
		if zoneID != "" {
			zoneIDCloudFare = zoneID
		}
		//get the email
		email := query.Get("email")
		if email != "" {
			emailCloudFare = email
		}
		//get the API Key
		apiKey := query.Get("apiKey")
		if apiKey != "" {
			apiKeyCloudFare = apiKey
		}
		//get the router URL
		routerURLParam := query.Get("routerURL")
		if routerURLParam != "" {
			routerURL = routerURLParam
		}
		//get the router admin
		routerAdminParam := query.Get("routerAdmin")
		if routerAdmin != "" {
			routerAdmin = routerAdminParam
		}
		//get the router password
		routerPasswordParam := query.Get("routerPassword")
		if routerPassword != "" {
			routerPassword = routerPasswordParam
		}

		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	// Handler if no route define
	route.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, "Page not found")
	})

}
func main() {
	// Start the cron job

	//install dependencies
	//clone the repository
	// clone https://github.com/tebeka/selenium

	tebekaCloneRepository := "https://github.com/tebeka/selenium"
	Pwd, err := os.Getwd()
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error getting the current working directory",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return
	}

	tebekaCloneRepositoryPath := Pwd + "/selenium"
	tebekaCloneRepositoryBranch := "master"

	// clone
	err = cloneRepository(tebekaCloneRepository, tebekaCloneRepositoryPath, tebekaCloneRepositoryBranch)
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error cloning the repository",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return
	}

	//check if dependencies is installed
	err = installDependencies(tebekaCloneRepositoryPath)
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error installing the dependencies",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return
	}

	cronJob()

	logger.SetLogConsole(logger.LogData{
		Message: "Server started successfully on port 1000",
		Level:   "INFO",
	})
	err = http.ListenAndServe("0.0.0.0:1000", limit(route))
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error starting the server",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return
	}

	//
	//// Start a Selenium WebDriver server instance
	//IPPublic := GetIPPublic()
	//updateDNSRecordCloudFare(IPPublic)

}

func installDependencies(tebekaCloneRepositoryPath string) error {

	//go run init.go --alsologtostderr  --download_browsers --download_latest
	dependenciesDir := tebekaCloneRepositoryPath + "/vendor"
	//check if the directory is exist so we dont need to install the dependencies
	//install the dependencies
	cmd := exec.Command("go", "run", "init.go", "--alsologtostderr", "--download_browsers", "--download_latest")
	cmd.Dir = dependenciesDir
	err := cmd.Run()
	if err != nil {
		return err
	}
	logger.SetLogConsole(logger.LogData{
		Message: "Dependencies installed successfully",
		Level:   "INFO",
	})
	return nil

}

func cloneRepository(repository string, path string, branch string) error {

	//clone the repository
	//clone

	//create the directory from pwd+/selenium
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path = pwd + "/selenium"
	//check if the directory is exist
	if _, err := os.Stat(path); os.IsExist(err) {
		//return because the directory is already exist
		logger.SetLogConsole(logger.LogData{
			Message: "Directory is already exist",
			Level:   "INFO",
		})
		return nil
	}

	//check if the directory is empty
	//dont clone if the directory is not empty
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	if len(files) > 0 {
		logger.SetLogConsole(logger.LogData{
			Message: "Directory is not empty, no need to clone the repository",
			Level:   "INFO",
		})
		return nil
	}

	//clone the repository
	cmd := exec.Command("git", "clone", repository, path)
	err = cmd.Run()
	if err != nil {
		return err
	}

	//checkout to the branch
	cmd = exec.Command("git", "checkout", branch)
	err = cmd.Run()
	if err != nil {
		return err
	}
	logger.SetLogConsole(logger.LogData{
		Message: "Repository cloned successfully",
		Level:   "INFO",
	})
	return nil

}

func GetIPPublic() string {
	selenium.SetDebug(false)
	// Start a Selenium WebDriver with headless Chrome

	Pwd, err := os.Getwd()
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error getting the IP Public",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return ""
	}

	vendorDepPath := Pwd + "/selenium/vendor"

	logger.SetLogConsole(logger.LogData{
		Message: "Starting Selenium WebDriver with headless Chrome in " + vendorDepPath + "/chromedriver",
		Level:   "INFO",
	})
	service, err := selenium.NewChromeDriverService(vendorDepPath+"/chromedriver", 4444)
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error starting Selenium WebDriver with headless Chrome in " + vendorDepPath + "/chromedriver",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return ""
	}
	defer service.Stop()

	// Connect to the WebDriver instance running locally and start a new browser session
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}
	chromeCaps := chrome.Capabilities{
		Path: vendorDepPath + "/chrome-linux/chrome",
		Args: []string{
			"--headless", // <<<
			"--no-sandbox",
			"--user-agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_2) AppleWebKit/604.4.7 (KHTML, like Gecko) Version/11.0.2 Safari/604.4.7",
		},
	}
	caps.AddChrome(chromeCaps)
	logger.SetLogConsole(logger.LogData{
		Message: "Connecting to the WebDriver instance running locally and start a new browser session",
		Level:   "INFO",
	})
	wd, err := selenium.NewRemote(caps, "")
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error connecting to the WebDriver instance running locally and start a new browser session",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return ""
	}
	defer wd.Quit()
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error closing the browser",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return ""
	}

	err = loginToRouter(err, wd)
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error logging in to the router",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return ""
	}

	wanIp, err := getWanIPAddress(wd)
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error getting WAN IP Address",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return ""
	}
	//check if the WAN IP Address is a public IP Address not 10.x.x.x
	var wanIpSplit []string
	wanIpSplit = strings.Split(wanIp, ".")
	for wanIpSplit[0] == "10" {
		logger.SetLogConsole(logger.LogData{
			Message: "CURRENT WAN IP Address:" + wanIp + " is not a public IP Address, refreshing the WAN IP Address",
			Level:   "INFO",
		})
		//refresh the WAN IP Address
		err = refreshTheIpAddress(wd)
		if err != nil {
			logger.SetLogConsole(logger.LogData{
				Message: "Error refreshing the WAN IP Address",
				CustomFields: logrus.Fields{
					"data": err,
				},
				Level: "ERROR",
			})
			return ""
		}

		//get the WAN IP Address
		wanIp, err = getWanIPAddress(wd)
		if err != nil {
			logger.SetLogConsole(logger.LogData{
				Message: "Error getting WAN IP Address",
				CustomFields: logrus.Fields{
					"data": err,
				},
				Level: "ERROR",
			})
			return ""
		}
		wanIpSplit = strings.Split(wanIp, ".")
	}
	logger.SetLogConsole(logger.LogData{
		Message: "WAN IP Address is a public IP Address:" + wanIp,
		Level:   "INFO",
	})
	//close the browser
	err = wd.Quit()
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error closing the browser",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return ""
	}
	var IpSplit []string
	IpSplit = strings.Split(wanIp, "/")
	return IpSplit[0]
}
func loginToRouter(err error, wd selenium.WebDriver) error {
	// Navigate to the login page
	err = wd.Get(routerURL)
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
	err = username.SendKeys(routerAdmin)
	if err != nil {
		return err
	}
	err = password.SendKeys(routerPassword)
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
	//sleep for 2 seconds
	//time.Sleep(time.Duration(100000) * time.Second)
	ipAdd, err := logicGetWanIPAddress(wd)
	if err != nil {
		return "", err
	}

	for strings.Contains(ipAdd, "0.0.0.0") {
		logger.SetLogConsole(logger.LogData{
			Message: "Waiting for WAN IP Address to be assigned",
			Level:   "INFO",
		})
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

	//change the value
	err = selectData.Click()
	if err != nil {
		return err
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
	apiKey := apiKeyCloudFare
	zoneID := zoneIDCloudFare
	// retrieve a list of DNS records for the zone
	dnsRecords, err := getDNSRecords(apiKey, zoneID)
	if err != nil {
		logger.SetLogConsole(logger.LogData{
			Message: "Error retrieving DNS records",
			CustomFields: logrus.Fields{
				"data": err,
			},
			Level: "ERROR",
		})
		return
	}
	//get the DNS record where the name is "lab.kasfi-dev.tech"
	for _, record := range dnsRecords {
		if record.Name == dnsRecordNameCloudFare || record.Name == "*."+dnsRecordNameCloudFare {
			err = checkToUpdateDnsRecordByName(IP, record, err, apiKey, zoneID)
			if err != nil {
				logger.SetLogConsole(logger.LogData{
					Message: "Error checking to update DNS record",
					CustomFields: logrus.Fields{
						"data": err,
					},
					Level: "ERROR",
				})
				return
			}
		}
	}

	//check if the WAN IP Address is the same as the DNS record

}

func checkToUpdateDnsRecordByName(IP string, record DNSRecord, err error, apiKey string, zoneID string) error {
	if IP == record.Content {
		logger.SetLogConsole(logger.LogData{
			Message: "WAN IP Address is the same as the DNS record (" + IP + "), no need to update the DNS record",
			Level:   "INFO",
		})
		return nil
	}
	logger.SetLogConsole(logger.LogData{
		Message: "WAN IP Address is not the same as the DNS record (" + record.Content + "), updating the DNS record " + record.Content + " and " + IP,
		Level:   "INFO",
	})
	//update the DNS record
	record.Content = IP
	err = updateDNSRecord(apiKey, zoneID, record)
	if err != nil {
		return err
	}

	logger.SetLogConsole(logger.LogData{
		Message: "DNS record updated successfully to " + IP + " for " + dnsRecordNameCloudFare,
		Level:   "INFO",
	})
	return nil
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
	req.Header.Set("X-Auth-Email", emailCloudFare)
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
	req.Header.Set("X-Auth-Email", emailCloudFare)
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
	logger.SetLogConsole(logger.LogData{
		Message: "DNS Record updated successfully with API cloud fare",
		Level:   "INFO",
	})
	return nil
}
