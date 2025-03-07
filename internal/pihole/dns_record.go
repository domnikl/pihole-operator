package pihole

import (
	"fmt"

	"github.com/domnikl/pihole-operator/api/v1alpha1"
)

type DNSRecord struct {
	Domain string
	Target string
	Type   v1alpha1.DNSRecordType
	TTL    *int32
}

func NewDNSRecordFromSpec(spec v1alpha1.DNSNameSpec) (*DNSRecord, error) {
	// TODO: targetIP is required for A records
	var target string

	if spec.Type == v1alpha1.A {
		if spec.TargetIP == nil {
			return nil, fmt.Errorf("targetIP is required for A records")
		}

		target = string(*spec.TargetIP)
	} else if spec.Type == v1alpha1.CName {
		if spec.Target == nil {
			return nil, fmt.Errorf("target is required for CNAME records")
		}

		target = string(*spec.Target)
	} else {
		return nil, fmt.Errorf("invalid DNS record type %s", spec.Type)
	}

	return &DNSRecord{
		Domain: spec.Domain,
		Target: target,
		Type:   spec.Type,
		TTL:    spec.TTL,
	}, nil
}

func (r *DNSRecord) Equals(other *DNSRecord) bool {
	if r.Domain != other.Domain {
		return false
	}
	if r.Target != other.Target {
		return false
	}
	if r.Type != other.Type {
		return false
	}
	if r.TTL != other.TTL {
		return false
	}

	return true
}
