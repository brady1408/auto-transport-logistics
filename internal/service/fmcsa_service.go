package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// FMCSA QCMobile API (https://mobile.fmcsa.dot.gov/QCDevsite/). Requires a
// registered WebKey; responses are wrapped in a top-level "content" element.
const fmcsaBaseURL = "https://mobile.fmcsa.dot.gov/qc/services"

var (
	// ErrFMCSANotConfigured is returned when no WebKey is set.
	ErrFMCSANotConfigured = errors.New("fmcsa: FMCSA_WEBKEY not configured")
	// ErrFMCSACarrierNotFound is returned when the API has no record for the number.
	ErrFMCSACarrierNotFound = errors.New("fmcsa: carrier not found")
	// ErrFMCSAInvalidNumber is returned when the DOT/MC number is empty or non-numeric.
	ErrFMCSAInvalidNumber = errors.New("fmcsa: invalid DOT/MC number")
)

type FMCSAService struct {
	httpClient *http.Client
	baseURL    string
	webKey     string
}

func NewFMCSAService(webKey string) *FMCSAService {
	return &FMCSAService{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    fmcsaBaseURL,
		webKey:     webKey,
	}
}

// Configured reports whether a WebKey is available.
func (s *FMCSAService) Configured() bool { return s.webKey != "" }

// CarrierVerification is the normalized result of an FMCSA carrier lookup.
type CarrierVerification struct {
	LegalName        string
	DBAName          string
	DOTNumber        int
	StatusCode       string // A = active, I = inactive
	AllowedToOperate bool
	OutOfService     bool
	OutOfServiceDate string

	SafetyRating     string // Satisfactory / Conditional / Unsatisfactory / Not Rated
	SafetyRatingDate string

	CommonAuthority   string // Active / Inactive / None
	ContractAuthority string
	BrokerAuthority   string

	BIPDInsuranceOnFile  string // formatted dollar amounts; empty when not reported
	BIPDInsuranceReqd    string
	CargoInsuranceOnFile string
	BondInsuranceOnFile  string

	TotalPowerUnits int
	TotalDrivers    int
	SnapshotDate    string
	VerifiedAt      time.Time

	// VerifiedNumber is the number actually looked up, prefixed with its
	// type, e.g. "DOT 44110" or "MC 123456".
	VerifiedNumber string
}

// Authorized reports whether the carrier is allowed to operate and not out of
// service.
func (v *CarrierVerification) Authorized() bool {
	return v.AllowedToOperate && !v.OutOfService
}

// Summary is a short status line persisted on the company record.
func (v *CarrierVerification) Summary() string {
	parts := make([]string, 0, 4)
	switch {
	case v.OutOfService:
		parts = append(parts, "Out of Service")
	case v.AllowedToOperate:
		parts = append(parts, "Authorized")
	default:
		parts = append(parts, "Not Authorized")
	}
	parts = append(parts, "Rating: "+v.SafetyRating)
	if v.BIPDInsuranceOnFile != "" {
		parts = append(parts, "BIPD "+v.BIPDInsuranceOnFile)
	}
	s := strings.Join(parts, " · ")
	if len(s) > 160 {
		s = s[:160]
	}
	return s
}

// QCMobile wire types. Elements are omitted by the API when they have no
// value, so everything optional is a pointer or zero-valued.
type fmcsaCarrier struct {
	LegalName        string          `json:"legalName"`
	DBAName          string          `json:"dbaName"`
	DOTNumber        int             `json:"dotNumber"`
	StatusCode       string          `json:"statusCode"`
	AllowedToOperate string          `json:"allowedToOperate"`
	OOSDate          *string         `json:"oosDate"`
	SafetyRating     *string         `json:"safetyRating"`
	SafetyRatingDate *string         `json:"safetyRatingDate"`
	CommonAuthority  *string         `json:"commonAuthorityStatus"`
	ContractAuth     *string         `json:"contractAuthorityStatus"`
	BrokerAuth       *string         `json:"brokerAuthorityStatus"`
	BIPDOnFile       *string         `json:"bipdInsuranceOnFile"`
	BIPDRequired     *string         `json:"bipdInsuranceRequired"`
	BIPDReqdAmount   *string         `json:"bipdRequiredAmount"`
	CargoOnFile      *string         `json:"cargoInsuranceOnFile"`
	BondOnFile       *string         `json:"bondInsuranceOnFile"`
	TotalPowerUnits  json.RawMessage `json:"totalPowerUnits"`
	TotalDrivers     json.RawMessage `json:"totalDrivers"`
	SnapshotDate     *string         `json:"snapshotDate"`
}

type fmcsaCarrierContent struct {
	Carrier *fmcsaCarrier `json:"carrier"`
}

type fmcsaCarrierEnvelope struct {
	Content *fmcsaCarrierContent `json:"content"`
}

type fmcsaDocketEnvelope struct {
	Content []fmcsaCarrierContent `json:"content"`
}

// VerifyByDOT looks up a carrier by USDOT number.
func (s *FMCSAService) VerifyByDOT(ctx context.Context, dotNumber string) (*CarrierVerification, error) {
	num, err := cleanFMCSANumber(dotNumber)
	if err != nil {
		return nil, err
	}
	body, err := s.get(ctx, "/carriers/"+num)
	if err != nil {
		return nil, err
	}
	var env fmcsaCarrierEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("fmcsa: decode response: %w", err)
	}
	if env.Content == nil || env.Content.Carrier == nil {
		return nil, ErrFMCSACarrierNotFound
	}
	v := normalizeCarrier(env.Content.Carrier)
	v.VerifiedNumber = "DOT " + num
	return v, nil
}

// VerifyByMC looks up a carrier by MC/docket number. When multiple carriers
// share a docket the first match is used.
func (s *FMCSAService) VerifyByMC(ctx context.Context, mcNumber string) (*CarrierVerification, error) {
	num, err := cleanFMCSANumber(mcNumber)
	if err != nil {
		return nil, err
	}
	body, err := s.get(ctx, "/carriers/docket-number/"+num)
	if err != nil {
		return nil, err
	}
	var env fmcsaDocketEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		// Some not-found responses use `"content": null` on the object form.
		var alt fmcsaCarrierEnvelope
		if err2 := json.Unmarshal(body, &alt); err2 == nil && alt.Content == nil {
			return nil, ErrFMCSACarrierNotFound
		}
		return nil, fmt.Errorf("fmcsa: decode response: %w", err)
	}
	for _, c := range env.Content {
		if c.Carrier != nil {
			v := normalizeCarrier(c.Carrier)
			v.VerifiedNumber = "MC " + num
			return v, nil
		}
	}
	return nil, ErrFMCSACarrierNotFound
}

// cleanFMCSANumber strips prefixes like "MC-" and separators, keeping digits.
func cleanFMCSANumber(v string) (string, error) {
	var b strings.Builder
	for _, r := range v {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "", ErrFMCSAInvalidNumber
	}
	return b.String(), nil
}

func (s *FMCSAService) get(ctx context.Context, path string) ([]byte, error) {
	if !s.Configured() {
		return nil, ErrFMCSANotConfigured
	}
	u := s.baseURL + path + "?webKey=" + url.QueryEscape(s.webKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("fmcsa: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		// A *url.Error's message embeds the full request URL, which carries
		// the webKey; surface only the operation and underlying cause.
		var ue *url.Error
		if errors.As(err, &ue) {
			return nil, fmt.Errorf("fmcsa: %s request failed: %w", ue.Op, ue.Err)
		}
		return nil, fmt.Errorf("fmcsa: request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("fmcsa: read response: %w", err)
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrFMCSACarrierNotFound
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("fmcsa: unexpected status %d", resp.StatusCode)
	}
	return body, nil
}

func normalizeCarrier(c *fmcsaCarrier) *CarrierVerification {
	v := &CarrierVerification{
		LegalName:            c.LegalName,
		DBAName:              c.DBAName,
		DOTNumber:            c.DOTNumber,
		StatusCode:           c.StatusCode,
		AllowedToOperate:     strings.EqualFold(c.AllowedToOperate, "Y"),
		OutOfService:         deref(c.OOSDate) != "",
		OutOfServiceDate:     deref(c.OOSDate),
		SafetyRating:         safetyRatingLabel(deref(c.SafetyRating)),
		SafetyRatingDate:     deref(c.SafetyRatingDate),
		CommonAuthority:      authorityLabel(deref(c.CommonAuthority)),
		ContractAuthority:    authorityLabel(deref(c.ContractAuth)),
		BrokerAuthority:      authorityLabel(deref(c.BrokerAuth)),
		BIPDInsuranceOnFile:  insuranceAmount(deref(c.BIPDOnFile)),
		BIPDInsuranceReqd:    insuranceAmount(requiredAmount(c)),
		CargoInsuranceOnFile: insuranceAmount(deref(c.CargoOnFile)),
		BondInsuranceOnFile:  insuranceAmount(deref(c.BondOnFile)),
		TotalPowerUnits:      rawInt(c.TotalPowerUnits),
		TotalDrivers:         rawInt(c.TotalDrivers),
		SnapshotDate:         deref(c.SnapshotDate),
		VerifiedAt:           time.Now(),
	}
	return v
}

// requiredAmount returns the numeric BIPD required amount only; the
// bipdInsuranceRequired Y/N flag is not a dollar figure and must not be
// rendered as one.
func requiredAmount(c *fmcsaCarrier) string {
	amt := strings.TrimSpace(deref(c.BIPDReqdAmount))
	if _, err := strconv.Atoi(amt); err != nil {
		return ""
	}
	return amt
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// rawInt tolerates numeric fields the API sometimes sends as strings.
func rawInt(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	s := strings.Trim(string(raw), `"`)
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func safetyRatingLabel(code string) string {
	switch strings.ToUpper(code) {
	case "S":
		return "Satisfactory"
	case "C":
		return "Conditional"
	case "U":
		return "Unsatisfactory"
	case "":
		return "Not Rated"
	default:
		return code
	}
}

func authorityLabel(code string) string {
	switch strings.ToUpper(code) {
	case "A":
		return "Active"
	case "I":
		return "Inactive"
	case "N", "":
		return "None"
	default:
		return code
	}
}

// insuranceAmount formats QCMobile insurance values, reported in thousands of
// dollars, as a dollar figure.
func insuranceAmount(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "N") {
		return ""
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return v
	}
	if n == 0 {
		return ""
	}
	return "$" + formatThousands(n*1000)
}

func formatThousands(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
