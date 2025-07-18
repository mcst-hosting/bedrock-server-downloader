package main

import (
	"egtyl.xyz/omnibill/tui"
	"egtyl.xyz/omnibill/tui/progress"
	"egtyl.xyz/shane/archiver"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"regexp"
)

const VERSION = "v1.0.6"
const CHROME_VERSIONS_URL = "https://versionhistory.googleapis.com/v1/chrome/platforms/win64/channels/stable/versions"
const MC_BEDROCK_DOWNLOAD_URL = "https://net-secondary.web.minecraft-services.net/api/v1.0/download/links"

var mcLinkRegex = regexp.MustCompile(`https://www.minecraft.net/bedrockdedicatedserver/bin-linux/bedrock-server-(.*).zip`)
var homeDir = os.Getenv("HOME")

var flagGrabChromeVersion bool
var flagDirectory string
var flagDoUpdate bool

func init() {
	flag.BoolVar(&flagGrabChromeVersion, "use-chrome-api", false, "uses the latest version of chrome from their API as the user agent.")
	flag.StringVar(&flagDirectory, "directory", "", "the directory to download and extract the server to.")
	flag.BoolVar(&flagDoUpdate, "update", false, "update mc bedrock server if new version is available.")
	flag.Parse()

	if len(flagDirectory) != 0 {
		homeDir = flagDirectory
	}
}

func main() {

	fmt.Println(tui.Format(tui.FgColorGrey, tui.FmtBold) + "[ " + tui.Format(tui.FgColorGold, tui.FmtBoldReset) + "Bedrock Server Downloader " + VERSION + tui.Format(tui.FgColorGrey, tui.FmtBold) + " ]" + tui.FmtReset)

	var currentVersion string
	if flagDoUpdate {
		curVersion, err := os.ReadFile(filepath.Join(homeDir, ".server_version"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Fatalln(err)
		}
		currentVersion = string(curVersion)
	}

	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalln(err)
	}

	httpClient := &http.Client{
		Jar: cookieJar,
		Transport: &headerTransport{
			T: http.DefaultTransport,
		},
	}

	mcVerReq, err := http.NewRequest("GET", MC_BEDROCK_DOWNLOAD_URL, nil)
	if err != nil {
		log.Fatalln(err)
	}

	mcVerResp, err := httpClient.Do(mcVerReq)
	if err != nil {
		log.Fatalln(err)
	}
	defer mcVerResp.Body.Close()

	var linkInfo linksResponse
	if err := json.NewDecoder(mcVerResp.Body).Decode(&linkInfo); err != nil {
		log.Fatalln(err)
	}

	var downloadInfo linkDownload
	for _, link := range linkInfo.Result.Links {
		if link.DownloadType == "serverBedrockLinux" {
			downloadInfo = link
			break
		}
	}

	var mcVerRegexMatch []string
	if len(mcLinkRegex.FindStringSubmatch(downloadInfo.DownloadURL)) > 0 {
		mcVerRegexMatch = mcLinkRegex.FindStringSubmatch(downloadInfo.DownloadURL)
	}

	gameVersion := mcVerRegexMatch[1]
	archiveDest := filepath.Join(homeDir, "bedrock_server_"+gameVersion+".zip")

	if flagDoUpdate && currentVersion == gameVersion {
		fmt.Println("No updates found.")
		return
	}

	fmt.Println(tui.Format(tui.FgColorGrey, tui.FmtBold) + "[ " + tui.Format(tui.FgColorGold, tui.FmtBoldReset) + "Found Version: " + gameVersion + tui.Format(tui.FgColorGrey, tui.FmtBold) + " ]" + tui.FmtReset)

	file, err := os.OpenFile(archiveDest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	downloadReq, err := http.NewRequest("GET", mcVerRegexMatch[0], nil)
	if err != nil {
		log.Fatalln(err)
	}

	downloadResp, err := httpClient.Do(downloadReq)
	if err != nil {
		log.Fatalln(err)
	}
	defer downloadResp.Body.Close()

	progressBar := progress.New(progress.ProgressInfo{
		Size: downloadResp.ContentLength,
		Desc: "Downloading Bedrock Server " + gameVersion,
	})

	if _, err := io.Copy(io.MultiWriter(file, progressBar), downloadResp.Body); err != nil {
		log.Fatalln(err)
	}
	if err := file.Close(); err != nil {
		log.Fatalln(err)
	}
	if err := progressBar.Close(); err != nil {
		log.Fatalln(err)
	}

	fmt.Println(tui.Format(tui.FgColorGrey, tui.FmtBold) + "[ " + tui.Format(tui.FgColorGold, tui.FmtBoldReset) + "Extracting server archive" + tui.Format(tui.FgColorGrey, tui.FmtBold) + " ]" + tui.FmtReset)

	if flagDoUpdate && currentVersion != gameVersion {
		if len(currentVersion) == 0 {
			if _, err := os.Stat(filepath.Join(homeDir, "bedrock_server")); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					if err := archiver.Extract(archiveDest, archiver.ExtractOptions{
						Overwrite: true,
						Folder:    homeDir,
					}); err != nil {
						log.Fatalln(err)
					}
				} else {
					log.Fatalln(err)
				}
			}
		} else {
			if err := archiver.ExtractFile(archiveDest, "bedrock_server", archiver.ExtractFileOptions{
				Overwrite: true,
				Folder:    homeDir,
			}); err != nil {
				log.Fatalln(err)
			}
		}
	} else {
		if err := archiver.Extract(archiveDest, archiver.ExtractOptions{
			Overwrite: true,
			Folder:    homeDir,
		}); err != nil {
			log.Fatalln(err)
		}
	}

	if err := os.WriteFile(filepath.Join(homeDir, ".server_version"), []byte(gameVersion), 0600); err != nil {
		log.Fatalln(err)
	}

	if err := os.Remove(archiveDest); err != nil {
		log.Fatalln(err)
	}

	fmt.Println(tui.Format(tui.FgColorGrey, tui.FmtBold) + "[ " + tui.Format(tui.FgColorGreen, tui.FmtBoldReset) + "Successfully downloaded bedrock server" + tui.Format(tui.FgColorGrey, tui.FmtBold) + " ]" + tui.FmtReset)

}
