package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Fixture mirrors the QCMobile /carriers/{dotNumber} response shape.
const fmcsaCarrierFixture = `{
  "content": {
    "carrier": {
      "allowedToOperate": "Y",
      "bipdInsuranceOnFile": "1000",
      "bipdInsuranceRequired": "Y",
      "bipdRequiredAmount": "750",
      "bondInsuranceOnFile": "0",
      "bondInsuranceRequired": "N",
      "brokerAuthorityStatus": "N",
      "cargoInsuranceOnFile": "100",
      "cargoInsuranceRequired": "N",
      "carrierOperation": {"carrierOperationCode": "A", "carrierOperationDesc": "Interstate"},
      "commonAuthorityStatus": "A",
      "contractAuthorityStatus": "I",
      "crashTotal": 0,
      "dbaName": "EXAMPLE AUTO TRANSPORT",
      "dotNumber": 44110,
      "ein": 123456789,
      "legalName": "EXAMPLE TRANSPORT LLC",
      "oosDate": null,
      "phyCity": "SALT LAKE CITY",
      "phyState": "UT",
      "safetyRating": "S",
      "safetyRatingDate": "2019-04-30",
      "snapshotDate": "2026-06-27",
      "statusCode": "A",
      "totalDrivers": 12,
      "totalPowerUnits": 9
    },
    "links": []
  },
  "retrievalDate": "2026-07-08T04:15:00.000+0000"
}`

const fmcsaDocketFixture = `{
  "content": [
    {
      "carrier": {
        "allowedToOperate": "N",
        "brokerAuthorityStatus": "N",
        "commonAuthorityStatus": "I",
        "contractAuthorityStatus": "N",
        "dbaName": null,
        "dotNumber": 99999,
        "legalName": "REVOKED CARRIER INC",
        "oosDate": "2024-11-02",
        "safetyRating": "U",
        "safetyRatingDate": "2023-01-15",
        "snapshotDate": "2026-06-27",
        "statusCode": "I",
        "totalDrivers": "3",
        "totalPowerUnits": "2"
      }
    }
  ]
}`

func newTestFMCSA(t *testing.T, handler http.HandlerFunc) (*FMCSAService, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	svc := NewFMCSAService("test-key")
	svc.baseURL = srv.URL
	svc.httpClient = srv.Client()
	return svc, srv
}

func TestVerifyByDOT(t *testing.T) {
	var gotPath, gotKey string
	svc, _ := newTestFMCSA(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.URL.Query().Get("webKey")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmcsaCarrierFixture))
	})

	v, err := svc.VerifyByDOT(context.Background(), " 44110 ")
	if err != nil {
		t.Fatalf("VerifyByDOT: %v", err)
	}
	if gotPath != "/carriers/44110" {
		t.Errorf("path = %q, want /carriers/44110", gotPath)
	}
	if gotKey != "test-key" {
		t.Errorf("webKey = %q, want test-key", gotKey)
	}
	if v.LegalName != "EXAMPLE TRANSPORT LLC" || v.DOTNumber != 44110 {
		t.Errorf("identity = %q/%d", v.LegalName, v.DOTNumber)
	}
	if !v.AllowedToOperate || v.OutOfService {
		t.Errorf("operating flags = allowed:%v oos:%v", v.AllowedToOperate, v.OutOfService)
	}
	if v.SafetyRating != "Satisfactory" || v.SafetyRatingDate != "2019-04-30" {
		t.Errorf("safety = %q %q", v.SafetyRating, v.SafetyRatingDate)
	}
	if v.CommonAuthority != "Active" || v.ContractAuthority != "Inactive" || v.BrokerAuthority != "None" {
		t.Errorf("authority = %q/%q/%q", v.CommonAuthority, v.ContractAuthority, v.BrokerAuthority)
	}
	if v.BIPDInsuranceOnFile != "$1,000,000" {
		t.Errorf("BIPD on file = %q, want $1,000,000", v.BIPDInsuranceOnFile)
	}
	if v.BIPDInsuranceReqd != "$750,000" {
		t.Errorf("BIPD required = %q, want $750,000", v.BIPDInsuranceReqd)
	}
	if v.CargoInsuranceOnFile != "$100,000" {
		t.Errorf("cargo = %q, want $100,000", v.CargoInsuranceOnFile)
	}
	// "0" on file means no coverage, not "$0" of coverage.
	if v.BondInsuranceOnFile != "" {
		t.Errorf("bond = %q, want empty for zero amount", v.BondInsuranceOnFile)
	}
	if v.VerifiedNumber != "DOT 44110" {
		t.Errorf("verified number = %q, want DOT 44110", v.VerifiedNumber)
	}
	if !v.Authorized() {
		t.Error("Authorized() = false, want true")
	}
	if v.TotalPowerUnits != 9 || v.TotalDrivers != 12 {
		t.Errorf("fleet = %d units %d drivers", v.TotalPowerUnits, v.TotalDrivers)
	}
	if got := v.Summary(); got != "Authorized · Rating: Satisfactory · BIPD $1,000,000" {
		t.Errorf("summary = %q", got)
	}
}

func TestVerifyByMC(t *testing.T) {
	var gotPath string
	svc, _ := newTestFMCSA(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmcsaDocketFixture))
	})

	v, err := svc.VerifyByMC(context.Background(), "MC-99999")
	if err != nil {
		t.Fatalf("VerifyByMC: %v", err)
	}
	if gotPath != "/carriers/docket-number/99999" {
		t.Errorf("path = %q", gotPath)
	}
	if v.AllowedToOperate {
		t.Error("expected not allowed to operate")
	}
	if !v.OutOfService || v.OutOfServiceDate != "2024-11-02" {
		t.Errorf("oos = %v %q", v.OutOfService, v.OutOfServiceDate)
	}
	if v.SafetyRating != "Unsatisfactory" {
		t.Errorf("safety = %q", v.SafetyRating)
	}
	// String-typed numerics from the API still parse.
	if v.TotalPowerUnits != 2 || v.TotalDrivers != 3 {
		t.Errorf("fleet = %d units %d drivers", v.TotalPowerUnits, v.TotalDrivers)
	}
	if v.VerifiedNumber != "MC 99999" {
		t.Errorf("verified number = %q, want MC 99999", v.VerifiedNumber)
	}
	if v.Authorized() {
		t.Error("Authorized() = true, want false")
	}
	if got := v.Summary(); got != "Out of Service · Rating: Unsatisfactory" {
		t.Errorf("summary = %q", got)
	}
}

func TestVerifyByDOTNotFound(t *testing.T) {
	svc, _ := newTestFMCSA(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content": null}`))
	})
	_, err := svc.VerifyByDOT(context.Background(), "1")
	if !errors.Is(err, ErrFMCSACarrierNotFound) {
		t.Errorf("err = %v, want ErrFMCSACarrierNotFound", err)
	}
}

func TestVerifyByMCNotFound(t *testing.T) {
	svc, _ := newTestFMCSA(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content": null}`))
	})
	_, err := svc.VerifyByMC(context.Background(), "12345")
	if !errors.Is(err, ErrFMCSACarrierNotFound) {
		t.Errorf("err = %v, want ErrFMCSACarrierNotFound", err)
	}
}

func TestVerifyUpstreamError(t *testing.T) {
	svc, _ := newTestFMCSA(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := svc.VerifyByDOT(context.Background(), "44110")
	if err == nil || errors.Is(err, ErrFMCSACarrierNotFound) {
		t.Errorf("err = %v, want upstream error", err)
	}
}

func TestVerifyNotConfigured(t *testing.T) {
	svc := NewFMCSAService("")
	if svc.Configured() {
		t.Error("Configured() = true for empty key")
	}
	_, err := svc.VerifyByDOT(context.Background(), "44110")
	if !errors.Is(err, ErrFMCSANotConfigured) {
		t.Errorf("err = %v, want ErrFMCSANotConfigured", err)
	}
}

func TestVerifyInvalidNumber(t *testing.T) {
	svc := NewFMCSAService("key")
	_, err := svc.VerifyByDOT(context.Background(), "abc")
	if !errors.Is(err, ErrFMCSAInvalidNumber) {
		t.Errorf("err = %v, want ErrFMCSAInvalidNumber", err)
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial tcp 127.0.0.1:443: connection refused")
}

func TestTransportErrorDoesNotLeakWebKey(t *testing.T) {
	const key = "super-secret-webkey"
	svc := NewFMCSAService(key)
	svc.httpClient = &http.Client{Transport: failingTransport{}}

	_, err := svc.VerifyByDOT(context.Background(), "44110")
	if err == nil {
		t.Fatal("expected transport error")
	}
	if strings.Contains(err.Error(), key) {
		t.Errorf("error message leaks webKey: %v", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error message lost underlying cause: %v", err)
	}
}

func TestInsuranceAmount(t *testing.T) {
	cases := map[string]string{
		"":     "",
		"N":    "",
		"0":    "",
		"000":  "",
		"750":  "$750,000",
		"1000": "$1,000,000",
	}
	for in, want := range cases {
		if got := insuranceAmount(in); got != want {
			t.Errorf("insuranceAmount(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRequiredAmountIgnoresFlag(t *testing.T) {
	flag := "Y"
	if got := requiredAmount(&fmcsaCarrier{BIPDRequired: &flag}); got != "" {
		t.Errorf("requiredAmount with only Y/N flag = %q, want empty", got)
	}
	amt := "750"
	if got := requiredAmount(&fmcsaCarrier{BIPDReqdAmount: &amt, BIPDRequired: &flag}); got != "750" {
		t.Errorf("requiredAmount = %q, want 750", got)
	}
	bad := "u"
	if got := requiredAmount(&fmcsaCarrier{BIPDReqdAmount: &bad}); got != "" {
		t.Errorf("requiredAmount with non-numeric amount = %q, want empty", got)
	}
}
