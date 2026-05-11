// Package tlscfg provides TLS certificate loading and optional self-signed auto-generation.
package tlscfg

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config controls certificate sources and auto-TLS material under DataDir.
type Config struct {
	CertFile       string
	KeyFile        string
	AutoCert       bool
	DataDir        string
	Organization   string
	CommonName     string
	ValidDays      int
	RenewWithinDays int
}

// Manager holds a TLS certificate and optional renewal loop.
type Manager struct {
	mu     sync.RWMutex
	cfg    Config
	cert   tls.Certificate
	cancel context.CancelFunc
}

const defaultValidDays = 365
const defaultRenewWithin = 30

// NewManager loads PEM cert/key from CertFile/KeyFile, or generates a self-signed RSA-4096
// certificate into DataDir/tls/ when AutoCert is true. Returns error if AutoCert is false and files are missing.
func NewManager(parent context.Context, cfg Config) (*Manager, error) {
	if cfg.ValidDays <= 0 {
		cfg.ValidDays = defaultValidDays
	}
	if cfg.RenewWithinDays <= 0 {
		cfg.RenewWithinDays = defaultRenewWithin
	}
	if strings.TrimSpace(cfg.Organization) == "" {
		cfg.Organization = "xray2wg"
	}
	if strings.TrimSpace(cfg.CommonName) == "" {
		cfg.CommonName = "xray2wg"
	}
	if cfg.AutoCert && cfg.DataDir != "" && (cfg.CertFile == "" || cfg.KeyFile == "") {
		cfg.CertFile = filepath.Join(cfg.DataDir, "tls", "server.crt")
		cfg.KeyFile = filepath.Join(cfg.DataDir, "tls", "server.key")
	}

	m := &Manager{cfg: cfg}
	if err := m.loadOrGenerate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	go m.renewalLoop(ctx)
	return m, nil
}

// Stop ends the renewal goroutine.
func (m *Manager) Stop() {
	if m != nil && m.cancel != nil {
		m.cancel()
	}
}

func (m *Manager) loadOrGenerate() error {
	if fileExists(m.cfg.CertFile) && fileExists(m.cfg.KeyFile) {
		cert, err := tls.LoadX509KeyPair(m.cfg.CertFile, m.cfg.KeyFile)
		if err != nil {
			return fmt.Errorf("load tls pair: %w", err)
		}
		m.mu.Lock()
		m.cert = cert
		m.mu.Unlock()
		return nil
	}
	if !m.cfg.AutoCert {
		return errors.New("tls: no certificate files and auto-cert disabled")
	}
	if err := m.generateAndSave(); err != nil {
		return err
	}
	cert, err := tls.LoadX509KeyPair(m.cfg.CertFile, m.cfg.KeyFile)
	if err != nil {
		return fmt.Errorf("load generated tls: %w", err)
	}
	m.mu.Lock()
	m.cert = cert
	m.mu.Unlock()
	return nil
}

func (m *Manager) generateAndSave() error {
	if err := os.MkdirAll(filepath.Dir(m.cfg.CertFile), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.cfg.KeyFile), 0o755); err != nil {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	notBefore := time.Now().UTC().Add(-1 * time.Hour)
	notAfter := notBefore.Add(time.Duration(m.cfg.ValidDays) * 24 * time.Hour)

	dns, ips := localSANs()
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{m.cfg.Organization},
			CommonName:     m.cfg.CommonName,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dns,
		IPAddresses:           ips,
	}

	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER := x509.MarshalPKCS1PrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(m.cfg.CertFile, certPEM, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(m.cfg.KeyFile, keyPEM, 0o600); err != nil {
		return err
	}
	return nil
}

func localSANs() (dns []string, ips []net.IP) {
	dns = []string{"localhost"}
	if lip := net.ParseIP("127.0.0.1"); lip != nil {
		if v4 := lip.To4(); v4 != nil {
			ips = append(ips, v4)
		}
	}
	if lip6 := net.ParseIP("::1"); lip6 != nil {
		ips = append(ips, lip6)
	}
	if h, err := os.Hostname(); err == nil && h != "" && h != "localhost" {
		dns = append(dns, h)
	}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				ips = append(ips, ip4)
			} else if ip.To16() != nil {
				ips = append(ips, ip)
			}
		}
	}
	return dns, ips
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func (m *Manager) leafNotAfter() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.cert.Certificate) == 0 {
		return time.Time{}
	}
	c, err := x509.ParseCertificate(m.cert.Certificate[0])
	if err != nil {
		return time.Time{}
	}
	return c.NotAfter
}

func (m *Manager) shouldRenew() bool {
	t := m.leafNotAfter()
	if t.IsZero() {
		return false
	}
	return time.Until(t) < time.Duration(m.cfg.RenewWithinDays)*24*time.Hour
}

func (m *Manager) renewalLoop(ctx context.Context) {
	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if m.shouldRenew() && m.cfg.AutoCert {
				_ = os.Remove(m.cfg.CertFile)
				_ = os.Remove(m.cfg.KeyFile)
				if err := m.generateAndSave(); err == nil {
					if cert, err := tls.LoadX509KeyPair(m.cfg.CertFile, m.cfg.KeyFile); err == nil {
						m.mu.Lock()
						m.cert = cert
						m.mu.Unlock()
					}
				}
			}
		}
	}
}

// GetCertificate implements tls.Config.GetCertificate.
func (m *Manager) GetCertificate(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c := m.cert
	return &c, nil
}

// TLSConfig returns a server-side tls.Config referencing this manager's certificate.
func (m *Manager) TLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return m.GetCertificate(chi)
		},
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}
}
