//go:build linux

package secrets

import (
	"errors"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	secretServiceName            = "org.freedesktop.secrets"
	secretServicePath            = dbus.ObjectPath("/org/freedesktop/secrets")
	secretServiceDefaultAlias    = dbus.ObjectPath("/org/freedesktop/secrets/aliases/default")
	secretServiceIface           = "org.freedesktop.Secret.Service"
	secretServiceCollectionIface = "org.freedesktop.Secret.Collection"
	secretServiceItemIface       = "org.freedesktop.Secret.Item"
	secretServicePromptIface     = "org.freedesktop.Secret.Prompt"
	secretServiceSessionIface    = "org.freedesktop.Secret.Session"
	secretServiceFallbackName    = "linux-secret-service+local-file-0600-fallback"
	secretServicePromptTimeout   = 90 * time.Second
)

type secretServiceSecret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

type secretServiceClient struct {
	conn    *dbus.Conn
	session dbus.ObjectPath
}

func backendStatus(s *Store) (Status, error) {
	if _, err := diskDir(s); err != nil {
		return Status{}, err
	}
	return Status{
		Backend: secretServiceFallbackName,
		Dir:     "org.freedesktop.secrets; fallback " + diskBackendName(),
	}, nil
}

func backendSave(s *Store, name string, secret string) error {
	client, err := newSecretServiceClient()
	if isSecretServiceUnavailable(err) {
		return diskSave(s, name, secret)
	}
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.save(name, secret); err != nil {
		if isSecretServiceUnavailable(err) {
			return diskSave(s, name, secret)
		}
		return err
	}
	_ = diskDelete(s, name)
	return nil
}

func backendLoad(s *Store, name string) (string, bool, error) {
	client, err := newSecretServiceClient()
	if isSecretServiceUnavailable(err) {
		return diskLoad(s, name)
	}
	if err != nil {
		return "", false, err
	}
	defer client.Close()
	secret, ok, err := client.load(name)
	if err != nil {
		if isSecretServiceUnavailable(err) {
			return diskLoad(s, name)
		}
		return "", false, err
	}
	if ok {
		return secret, true, nil
	}
	return diskLoad(s, name)
}

func backendDelete(s *Store, name string) error {
	client, err := newSecretServiceClient()
	if isSecretServiceUnavailable(err) {
		return diskDelete(s, name)
	}
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.delete(name); err != nil && !isSecretServiceUnavailable(err) {
		return err
	}
	return diskDelete(s, name)
}

func newSecretServiceClient() (*secretServiceClient, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	obj := conn.Object(secretServiceName, secretServicePath)
	var output dbus.Variant
	var session dbus.ObjectPath
	if err := obj.Call(secretServiceIface+".OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&output, &session); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if session == "" || session == dbus.ObjectPath("/") {
		_ = conn.Close()
		return nil, errors.New("Secret Service returned an empty session")
	}
	return &secretServiceClient{conn: conn, session: session}, nil
}

func (c *secretServiceClient) Close() {
	if c == nil || c.conn == nil {
		return
	}
	if c.session != "" && c.session != dbus.ObjectPath("/") {
		_ = c.conn.Object(secretServiceName, c.session).Call(secretServiceSessionIface+".Close", 0).Err
	}
	_ = c.conn.Close()
}

func (c *secretServiceClient) save(name string, secret string) error {
	account := safeName(name)
	properties := map[string]dbus.Variant{
		secretServiceItemIface + ".Label":      dbus.MakeVariant("RecordingFreedom OCR Translation - " + account),
		secretServiceItemIface + ".Attributes": dbus.MakeVariant(secretAttributes(account)),
	}
	payload := secretServiceSecret{
		Session:     c.session,
		Parameters:  []byte{},
		Value:       []byte(secret),
		ContentType: "text/plain; charset=utf-8",
	}
	var item dbus.ObjectPath
	var prompt dbus.ObjectPath
	err := c.conn.Object(secretServiceName, secretServiceDefaultAlias).
		Call(secretServiceCollectionIface+".CreateItem", 0, properties, payload, true).
		Store(&item, &prompt)
	if err != nil {
		return err
	}
	return c.prompt(prompt)
}

func (c *secretServiceClient) load(name string) (string, bool, error) {
	paths, err := c.searchUnlocked(name)
	if err != nil || len(paths) == 0 {
		return "", false, err
	}
	secrets, err := c.getSecrets(paths)
	if err != nil {
		return "", false, err
	}
	for _, path := range paths {
		payload, ok := secrets[path]
		if !ok {
			continue
		}
		secret := strings.TrimSpace(string(payload.Value))
		if secret != "" {
			return secret, true, nil
		}
	}
	return "", false, nil
}

func (c *secretServiceClient) delete(name string) error {
	paths, err := c.searchUnlocked(name)
	if err != nil {
		return err
	}
	for _, path := range paths {
		var prompt dbus.ObjectPath
		err := c.conn.Object(secretServiceName, path).
			Call(secretServiceItemIface+".Delete", 0).
			Store(&prompt)
		if err != nil {
			return err
		}
		if err := c.prompt(prompt); err != nil {
			return err
		}
	}
	return nil
}

func (c *secretServiceClient) searchUnlocked(name string) ([]dbus.ObjectPath, error) {
	var unlocked []dbus.ObjectPath
	var locked []dbus.ObjectPath
	err := c.conn.Object(secretServiceName, secretServicePath).
		Call(secretServiceIface+".SearchItems", 0, secretAttributes(safeName(name))).
		Store(&unlocked, &locked)
	if err != nil {
		return nil, err
	}
	if len(locked) > 0 {
		if err := c.unlock(locked); err != nil {
			return nil, err
		}
		unlocked = append(unlocked, locked...)
	}
	return unlocked, nil
}

func (c *secretServiceClient) unlock(paths []dbus.ObjectPath) error {
	var unlocked []dbus.ObjectPath
	var prompt dbus.ObjectPath
	err := c.conn.Object(secretServiceName, secretServicePath).
		Call(secretServiceIface+".Unlock", 0, paths).
		Store(&unlocked, &prompt)
	if err != nil {
		return err
	}
	return c.prompt(prompt)
}

func (c *secretServiceClient) getSecrets(paths []dbus.ObjectPath) (map[dbus.ObjectPath]secretServiceSecret, error) {
	result := map[dbus.ObjectPath]secretServiceSecret{}
	if len(paths) == 0 {
		return result, nil
	}
	err := c.conn.Object(secretServiceName, secretServicePath).
		Call(secretServiceIface+".GetSecrets", 0, paths, c.session).
		Store(&result)
	return result, err
}

func (c *secretServiceClient) prompt(prompt dbus.ObjectPath) error {
	if prompt == "" || prompt == dbus.ObjectPath("/") {
		return nil
	}
	ch := make(chan *dbus.Signal, 4)
	c.conn.Signal(ch)
	defer c.conn.RemoveSignal(ch)
	options := []dbus.MatchOption{
		dbus.WithMatchObjectPath(prompt),
		dbus.WithMatchInterface(secretServicePromptIface),
		dbus.WithMatchMember("Completed"),
	}
	if err := c.conn.AddMatchSignal(options...); err != nil {
		return err
	}
	defer func() { _ = c.conn.RemoveMatchSignal(options...) }()
	if err := c.conn.Object(secretServiceName, prompt).Call(secretServicePromptIface+".Prompt", 0, "").Err; err != nil {
		return err
	}
	timer := time.NewTimer(secretServicePromptTimeout)
	defer timer.Stop()
	for {
		select {
		case signal := <-ch:
			if signal == nil || signal.Path != prompt || signal.Name != secretServicePromptIface+".Completed" {
				continue
			}
			if len(signal.Body) > 0 {
				if dismissed, ok := signal.Body[0].(bool); ok && dismissed {
					return errors.New("Secret Service prompt was dismissed")
				}
			}
			return nil
		case <-timer.C:
			return errors.New("Secret Service prompt timed out")
		}
	}
}

func secretAttributes(account string) map[string]string {
	return map[string]string{
		"application": "RecordingFreedom",
		"component":   "ocr-translation",
		"account":     account,
	}
}

func isSecretServiceUnavailable(err error) bool {
	if err == nil {
		return false
	}
	var dbusErr dbus.Error
	if errors.As(err, &dbusErr) {
		switch dbusErr.Name {
		case "org.freedesktop.DBus.Error.ServiceUnknown",
			"org.freedesktop.DBus.Error.NameHasNoOwner",
			"org.freedesktop.DBus.Error.Spawn.ChildExited",
			"org.freedesktop.DBus.Error.Spawn.ExecFailed":
			return true
		}
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "unable to autolaunch a dbus-daemon") ||
		strings.Contains(text, "dbus-launch") ||
		strings.Contains(text, "no such file or directory") ||
		strings.Contains(text, "org.freedesktop.dbus.error.serviceunknown") ||
		strings.Contains(text, "name has no owner")
}
