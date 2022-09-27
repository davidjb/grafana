package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/build/config"
	"github.com/grafana/grafana/pkg/build/gcloud"
	"github.com/grafana/grafana/pkg/build/gcloud/storage"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// GrafanaCom implements the sub-command "grafana-com".
func GrafanaCom(c *cli.Context) error {
	bucketStr := c.String("src-bucket")
	edition := config.Edition(c.String("edition"))

	if err := gcloud.ActivateServiceAccount(); err != nil {
		return fmt.Errorf("couldn't activate service account, err: %w", err)
	}

	metadata, err := GenerateMetadata(c)
	if err != nil {
		return err
	}

	releaseMode, err := metadata.GetReleaseMode()
	if err != nil {
		return err
	}

	version := metadata.GrafanaVersion
	if releaseMode.Mode == config.Cronjob {
		gcs, err := storage.New()
		if err != nil {
			return err
		}
		bucket := gcs.Bucket(bucketStr)
		latestMainVersion, err := storage.GetLatestMainBuild(c.Context, bucket, filepath.Join(string(edition), "main"))
		if err != nil {
			return err
		}
		version = latestMainVersion
	}

	dryRun := c.Bool("dry-run")
	simulateRelease := c.Bool("simulate-release")
	// Test release mode and dryRun imply simulateRelease
	if releaseMode.IsTest || dryRun {
		simulateRelease = true
	}

	grafanaAPIKey := strings.TrimSpace(os.Getenv("GRAFANA_COM_API_KEY"))
	if grafanaAPIKey == "" {
		return cli.NewExitError("the environment variable GRAFANA_COM_API_KEY must be set", 1)
	}
	whatsNewURL, releaseNotesURL, err := getReleaseURLs()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	// TODO: Verify config values
	cfg := PublishConfig{
		Config: config.Config{
			Version: version,
		},
		Edition:         edition,
		ReleaseMode:     releaseMode,
		GrafanaAPIKey:   grafanaAPIKey,
		WhatsNewURL:     whatsNewURL,
		ReleaseNotesURL: releaseNotesURL,
		DryRun:          dryRun,
		TTL:             c.String("ttl"),
		SimulateRelease: simulateRelease,
	}

	if err := publishPackages(cfg); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	log.Info().Msg("Successfully published packages to grafana.com!")
	return nil
}

func getReleaseURLs() (string, string, error) {
	type grafanaConf struct {
		WhatsNewURL     string `json:"whatsNewUrl"`
		ReleaseNotesURL string `json:"releaseNotesUrl"`
	}
	type packageConf struct {
		Grafana grafanaConf `json:"grafana"`
	}

	pkgB, err := ioutil.ReadFile("package.json")
	if err != nil {
		return "", "", fmt.Errorf("failed to read package.json: %w", err)
	}

	var pconf packageConf
	if err := json.Unmarshal(pkgB, &pconf); err != nil {
		return "", "", fmt.Errorf("failed to decode package.json: %w", err)
	}
	if _, err := url.ParseRequestURI(pconf.Grafana.WhatsNewURL); err != nil {
		return "", "", fmt.Errorf("grafana.whatsNewUrl is invalid in package.json: %q", pconf.Grafana.WhatsNewURL)
	}
	if _, err := url.ParseRequestURI(pconf.Grafana.ReleaseNotesURL); err != nil {
		return "", "", fmt.Errorf("grafana.releaseNotesUrl is invalid in package.json: %q",
			pconf.Grafana.ReleaseNotesURL)
	}

	return pconf.Grafana.WhatsNewURL, pconf.Grafana.ReleaseNotesURL, nil
}

// publishPackages publishes packages to grafana.com.
func publishPackages(cfg PublishConfig) error {
	log.Info().Msgf("Publishing Grafana packages, version %s, %s edition, %s mode, dryRun: %v, simulating: %v...",
		cfg.Version, cfg.Edition, cfg.ReleaseMode.Mode, cfg.DryRun, cfg.SimulateRelease)

	versionStr := fmt.Sprintf("v%s", cfg.Version)
	log.Info().Msgf("Creating release %s at grafana.com...", versionStr)

	var sfx string
	var pth string
	switch cfg.Edition {
	case config.EditionOSS:
		pth = "oss"
	case config.EditionEnterprise:
		pth = "enterprise"
		sfx = EnterpriseSfx
	default:
		return fmt.Errorf("unrecognized edition %q", cfg.Edition)
	}

	switch cfg.ReleaseMode.Mode {
	case config.MainMode, config.CustomMode, config.CronjobMode:
		pth = path.Join(pth, MainFolder)
	default:
		pth = path.Join(pth, ReleaseFolder)
	}

	product := fmt.Sprintf("grafana%s", sfx)
	pth = path.Join(pth, product)
	baseArchiveURL := fmt.Sprintf("https://dl.grafana.com/%s", pth)

	builds := []buildRepr{}
	for _, ba := range ArtifactConfigs {
		u := ba.GetURL(baseArchiveURL, cfg)

		shaURL := fmt.Sprintf("%s.sha256", u)
		resp, err := http.Get(shaURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("failed downloading %s: %s", u, resp.Status)
		}
		sha256, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		builds = append(builds, buildRepr{
			OS:     ba.Os,
			URL:    u,
			SHA256: string(sha256),
			Arch:   ba.Arch,
		})
	}

	r := releaseRepr{
		Version:     cfg.Version,
		ReleaseDate: time.Now().UTC(),
		Builds:      builds,
		Stable:      cfg.ReleaseMode.Mode == config.TagMode,
		Beta:        cfg.ReleaseMode.IsBeta,
		Nightly:     cfg.ReleaseMode.Mode == config.CronjobMode,
	}
	if cfg.ReleaseMode.Mode == config.TagMode || r.Beta {
		r.WhatsNewURL = cfg.WhatsNewURL
		r.ReleaseNotesURL = cfg.ReleaseNotesURL
	}

	if err := postRequest(cfg, "versions", r, fmt.Sprintf("create release %s", r.Version)); err != nil {
		return err
	}

	if err := postRequest(cfg, fmt.Sprintf("versions/%s", cfg.Version), r,
		fmt.Sprintf("update release %s", cfg.Version)); err != nil {
		return err
	}

	for _, b := range r.Builds {
		if err := postRequest(cfg, fmt.Sprintf("versions/%s/packages", cfg.Version), b,
			fmt.Sprintf("create build %s %s", b.OS, b.Arch)); err != nil {
			return err
		}
		if err := postRequest(cfg, fmt.Sprintf("versions/%s/packages/%s/%s", cfg.Version, b.Arch, b.OS), b,
			fmt.Sprintf("update build %s %s", b.OS, b.Arch)); err != nil {
			return err
		}
	}

	return nil
}

func postRequest(cfg PublishConfig, pth string, obj interface{}, descr string) error {
	var sfx string
	switch cfg.Edition {
	case config.EditionOSS:
	case config.EditionEnterprise:
		sfx = EnterpriseSfx
	default:
		return fmt.Errorf("unrecognized edition %q", cfg.Edition)
	}
	product := fmt.Sprintf("grafana%s", sfx)

	jsonB, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to JSON encode release: %w", err)
	}

	u := fmt.Sprintf("https://grafana.com/api/%s/%s", product, pth)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(jsonB))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.GrafanaAPIKey))
	req.Header.Add("Content-Type", "application/json")

	log.Info().Msgf("Posting to grafana.com API, %s - JSON: %s", u, string(jsonB))
	if cfg.SimulateRelease {
		log.Info().Msgf("Only simulating request")
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed posting to %s (%s): %s", u, descr, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if strings.Contains(string(body), "already exists") || strings.Contains(string(body), "Nothing to update") {
			log.Info().Msgf("Already exists: %s", descr)
			return nil
		}

		return fmt.Errorf("failed posting to %s (%s): %s", u, descr, resp.Status)
	}

	log.Info().Msgf("Successfully posted to grafana.com API, %s", u)

	return nil
}

type buildRepr struct {
	OS     string `json:"os"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Arch   string `json:"arch"`
}

type releaseRepr struct {
	Version         string      `json:"version"`
	ReleaseDate     time.Time   `json:"releaseDate"`
	Stable          bool        `json:"stable"`
	Beta            bool        `json:"beta"`
	Nightly         bool        `json:"nightly"`
	WhatsNewURL     string      `json:"whatsNewUrl"`
	ReleaseNotesURL string      `json:"releaseNotesUrl"`
	Builds          []buildRepr `json:"-"`
}
