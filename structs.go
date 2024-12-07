package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type chromeVerList struct {
	Versions []chromeVer `json:"versions"`
}

type chromeVer struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type headerTransport struct {
	T http.RoundTripper
}

var latestChromeVer string
var latestChromeMajor string

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("priority", "u=0, i")
	req.Header.Set("sec-ch-prefers-color-scheme", "dark")
	req.Header.Set("sec-ch-ua-arch", `"x86"`)
	req.Header.Set("sec-ch-ua-bitness", `"64"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-model", `""`)
	req.Header.Set("sec-ch-ua-platform", `"Linux"`)
	req.Header.Set("sec-ch-ua-platform-version", `"6.9.3"`)
	req.Header.Set("sec-ch-ua-wow64", "?0")
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-user", "?1")
	req.Header.Set("upgrade-insecure-requests", "1")

	if flagGrabChromeVersion {
		if latestChromeVer != "" && latestChromeMajor != "" {
			chromeVerResp, err := http.Get(CHROME_VERSIONS_URL)
			if err != nil {
				log.Fatalln(err)
			}
			defer chromeVerResp.Body.Close()

			var chromeVers chromeVerList
			if err := json.NewDecoder(chromeVerResp.Body).Decode(&chromeVers); err != nil {
				log.Fatalln(err)
			}

			latestChromeVer = chromeVers.Versions[0].Version
			latestChromeMajor = strings.Split(latestChromeVer, ".")[0]
		}

		req.Header.Set("sec-ch-ua-full-version", fmt.Sprintf(`"%s"`, latestChromeVer))
		req.Header.Set("sec-ch-ua-full-version-list", fmt.Sprintf(`"Not/A)Brand";v="8.0.0.0", "Chromium";v="%s"`, latestChromeVer))
		req.Header.Set("sec-ch-ua", fmt.Sprintf(`"Not/A)Brand";v="8", "Chromium";v="%s"`, latestChromeMajor))
		req.Header.Set("user-agent", fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", latestChromeVer))
	} else {
		req.Header.Set("sec-ch-ua-full-version", "131.0.6778.86")
		req.Header.Set("sec-ch-ua-full-version-list", `"Not/A)Brand";v="8.0.0.0", "Chromium";v="131.0.6778.86"`)
		req.Header.Set("sec-ch-ua", `"Not/A)Brand";v="8", "Chromium";v="131"`)
		req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.6778.86 Safari/537.36")
	}

	return t.T.RoundTrip(req)
}
