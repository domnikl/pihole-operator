package pihole

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/domnikl/pihole-operator/api/v1alpha1"
)

type DNSRecord struct {
	Domain string
	Target string
	Type   v1alpha1.DNSRecordType
	TTL    *int32
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
	var records []DNSRecord

	// A Records
	aRecords, err := p.getARecords()
	if err != nil {
		return nil, err
	}

	records = append(records, aRecords...)

	// CNAME Records
	cnameRecords, err := p.getCNames()
	if err != nil {
		return nil, err
	}

	records = append(records, cnameRecords...)

	return records, nil
}

func (p *PiHole) getCNames() ([]DNSRecord, error) {
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
		var ttl *int32
		parts := strings.Split(record, ",")

		// CNAME records have a TTL
		if len(parts) == 3 {
			x, err := strconv.ParseInt(parts[2], 10, 32)
			if err != nil {
				return nil, err
			}

			a := int32(x)
			ttl = &a
		}

		recordsList = append(recordsList, DNSRecord{
			Domain: parts[0],
			Target: parts[1],
			TTL:    ttl,
			Type:   v1alpha1.CName,
		})
	}

	return recordsList, nil
}

func (p *PiHole) getARecords() ([]DNSRecord, error) {
	resp, err := p.doAuthenticatedRequest(http.MethodGet, "/config/dns/hosts", nil)
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
				Hosts []string `json:"hosts"`
			} `json:"dns"`
		} `json:"config"`
	}

	var records response
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, err
	}

	var recordsList []DNSRecord
	for _, record := range records.Config.DNS.Hosts {
		parts := strings.Split(record, " ")

		recordsList = append(recordsList, DNSRecord{
			Target: parts[0],
			Domain: parts[1],
			Type:   v1alpha1.A,
		})
	}

	return recordsList, nil
}

func (p *PiHole) CreateDNSCNAMERecord(domain string, target string, ttl *int32) error {
	record := fmt.Sprintf("%s,%s", domain, target)
	if ttl != nil {
		record = fmt.Sprintf("%s,%s,%d", domain, target, *ttl)
	}

	resp, err := p.doAuthenticatedRequest(http.MethodPut, fmt.Sprintf("/config/dns/cnameRecords/%s", record), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create DNS CNAME record with status code %d", resp.StatusCode)
	}

	return nil
}

func (p *PiHole) CreateDNSARecord(domain string, ip v1alpha1.IPAddressStr) error {
	resp, err := p.doAuthenticatedRequest(http.MethodPut, fmt.Sprintf("/config/dns/hosts/%s %s", ip, domain), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create DNS A record with status code %d", resp.StatusCode)
	}

	return nil
}

func (p *PiHole) DeleteDNSRecord(record DNSRecord) error {
	var domain, path string
	if record.Type == v1alpha1.A {
		domain = fmt.Sprintf("%s %s", record.Target, record.Domain)
		path = "hosts"
	} else if record.Type == v1alpha1.CName {
		if record.TTL != nil {
			domain = fmt.Sprintf("%s,%s,%d", record.Domain, record.Target, *record.TTL)
		} else {
			domain = fmt.Sprintf("%s,%s", record.Domain, record.Target)
		}

		path = "cnameRecords"
	} else {
		return fmt.Errorf("invalid DNS record type %s", record.Type)
	}

	resp, err := p.doAuthenticatedRequest(http.MethodDelete, fmt.Sprintf("/config/dns/%s/%s", path, domain), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		fmt.Printf("failed to delete DNS record: %d - %s/%s", resp.StatusCode, path, domain)

		return fmt.Errorf("failed to delete DNS record with status code %d", resp.StatusCode)
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
