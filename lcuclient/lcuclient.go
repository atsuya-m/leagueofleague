package lcuclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/mitchellh/go-ps"
	"github.com/shirou/gopsutil/process"
)

type LCUClient struct {
	client *http.Client
	config *leagueConfig
}

func NewClient() (*LCUClient, error) {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	certs, err := ioutil.ReadFile("./riotgames.pem")
	if err != nil {
		return nil, err
	}

	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		return nil, errors.New("Coudn't install certification specified.")
	}

	config, err := getLeagueCmd()
	if err != nil {
		return nil, err
	}

	tlsconfig := &tls.Config{RootCAs: rootCAs}
	tr := &http.Transport{TLSClientConfig: tlsconfig}
	client := &http.Client{Transport: tr}

	log.Printf("PORT: %d; PASS: %s; ENCODE: %s", config.port, config.token, config.getEncodedToken())

	return &LCUClient{
		client: client,
		config: config,
	}, nil
}

func (c *LCUClient) Get(url string, dist interface{}) error {
	request, _ := http.NewRequest("GET", "https://127.0.0.1:"+strconv.Itoa(c.config.port)+url, nil)
	request.Header.Set("Authorization", "Basic "+c.config.getEncodedToken())

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}

	byteArray, _ := ioutil.ReadAll(response.Body)
	fmt.Println(string(byteArray))
	err = json.NewDecoder(response.Body).Decode(dist)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func getLeagueCmd() (*leagueConfig, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}

	var pid int
	for _, p := range processes {
		if p.Executable() == "LeagueClientUx.exe" {
			pid = p.PPid()
		}
	}
	if pid == 0 {
		return nil, errors.New("Couldn't find LeagueClient")
	}

	var proc *process.Process
	proc, err = process.NewProcess(int32(pid))
	if err != nil {
		return nil, err
	}
	cmd, err := proc.CmdlineSlice()
	if err != nil {
		return nil, err
	}

	lc := &leagueConfig{}
	for _, c := range cmd {
		switch true {
		case strings.HasPrefix(c, "--riotclient-auth-token"):
			lc.token = strings.Split(c, "=")[1]
		case strings.HasPrefix(c, "--riotclient-app-port"):
			lc.port, _ = strconv.Atoi(strings.Split(c, "=")[1])
		case strings.HasPrefix(c, "--region"):
			lc.region = strings.Split(c, "=")[1]
		}
	}

	return lc, nil
}

type leagueConfig struct {
	token  string
	port   int
	region string
}

func (c *leagueConfig) getEncodedToken() string {
	return base64.StdEncoding.EncodeToString([]byte("riot:" + c.token))
}
