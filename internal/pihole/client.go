package pihole

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type DNSRecordType string

const (
	CName DNSRecordType = "CNAME"
	A     DNSRecordType = "A"
)

type DNSRecord struct {
	Domain string
	Target string
	Type   DNSRecordType
}

type PiHole struct {
	// URL is the URL of the PiHole API
	URL string
	// AppPassword is the password to authenticate against the PiHole API
	AppPassword string
	sid         string
}

func NewPiHole(url string, appPassword string) *PiHole {
	return &PiHole{
		URL:         url,
		AppPassword: appPassword,
	}
}

func (p *PiHole) GetDNSRecords() ([]DNSRecord, error) {
	resp, err := p.doAuthenticatedRequest(http.MethodGet, "/config/dns/cnameRecords", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get DNS records with status code %d", resp.StatusCode)
	}

	type response struct {
		Config struct {
			DNS struct {
				CNameRecords []string `json:"cnameRecords"`
			} `json:"dns"`
		} `json:"config"`
	}

	var records response
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, err
	}

	var recordsList []DNSRecord
	for _, record := range records.Config.DNS.CNameRecords {
		parts := strings.Split(record, ",")

		recordsList = append(recordsList, DNSRecord{
			Domain: parts[0],
			Target: parts[1],
			Type:   CName,
		})
	}

	return recordsList, nil
}

func (p *PiHole) CreateDNSRecord(domain string, ip string) error {
	resp, err := p.doAuthenticatedRequest(http.MethodPut, fmt.Sprintf("/config/dns/cnameRecords/%s,%s", domain, ip), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create DNS record with status code %d", resp.StatusCode)
	}

	return nil
}

func (p *PiHole) authenticate() error {
	type authRequest struct {
		Password string `json:"password"`
	}

	request := authRequest{
		Password: p.AppPassword,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	resp, err := p.doRequest(http.MethodPost, "/auth", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status code %d", resp.StatusCode)
	}

	type authResponse struct {
		Session struct {
			Valid bool   `json:"valid"`
			SID   string `json:"sid"`
		} `json:"session"`
	}

	var response authResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.Session.Valid {
		return fmt.Errorf("authentication failed %v", response)
	}

	p.sid = response.Session.SID

	return nil
}

func (p *PiHole) doAuthenticatedRequest(method string, path string, body []byte) (*http.Response, error) {
	if p.sid == "" {
		if err := p.authenticate(); err != nil {
			return nil, err
		}
	}

	return p.doRequest(method, path, body)
}

func (p *PiHole) doRequest(method string, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, p.URL+path, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		log.Fatal(err)
	}

	if p.sid != "" {
		req.Header.Add("sid", p.sid)
	}

	client := &http.Client{}
	return client.Do(req)
}

func (p *PiHole) Close() error {
	resp, err := p.doAuthenticatedRequest(http.MethodDelete, "/auth", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to close session with status code %d", resp.StatusCode)
	}

	log.Println("PiHole session closed")

	return nil
}
