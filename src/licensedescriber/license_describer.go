package licensedescriber

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type LicenseSummary struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type TLDRLegalLicenseDescriber struct {
}

func NewTLDRDescriber() *TLDRLegalLicenseDescriber {
	return &TLDRLegalLicenseDescriber{}
}

// TODO: implement
func (d *TLDRLegalLicenseDescriber) Describe(license string) (string, error) {
	if true {
		return "", nil
	}

	summary, err := d.GetLicenseSummary(license)
	if err != nil {
		return "", fmt.Errorf("failed to fetch license summary: %w", err)
	}

	return summary.Summary, nil

}
func (d *TLDRLegalLicenseDescriber) GetLicenseSummary(license string) (*LicenseSummary, error) {
	url := fmt.Sprintf("https://api.tldrlegal.com/v1/licenses/%s", license)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch license summary: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var summary LicenseSummary
	if err := json.Unmarshal(body, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}
