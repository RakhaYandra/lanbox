package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	shareFrom    string
	shareToken   string
	sharePin     string
	shareExpires string
	shareNeedPIN bool
	shareUpload  bool
)

var shareCmd = &cobra.Command{
	Use:   "share <path>",
	Short: "Create an expiring share link for a server file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if shareFrom == "" {
			return fmt.Errorf("missing --from (e.g. --from 192.168.1.10:8080)")
		}
		dur, err := time.ParseDuration(shareExpires)
		if err != nil || dur < time.Minute || dur > 24*time.Hour {
			return fmt.Errorf("invalid --expires %q (try 30m, max 24h)", shareExpires)
		}
		token := shareToken
		if token == "" {
			token = os.Getenv("LANBOX_TOKEN")
		}
		body, _ := json.Marshal(map[string]any{
			"path":            args[0],
			"expires_minutes": int(dur.Minutes()),
			"pin_required":    shareNeedPIN,
			"allow_upload":    shareUpload,
		})
		req, err := http.NewRequest("POST", apiURL(shareFrom, "/api/v1/shares"), bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		setAuth(req, token, pinFor(sharePin))
		resp, err := tlsClient.Do(req)
		if err != nil {
			return fmt.Errorf("cannot connect — is lanbox serve running? %w", err)
		}
		defer resp.Body.Close()
		var result struct {
			ShareToken  string `json:"share_token"`
			URL         string `json:"url"`
			ExpiresAt   int64  `json:"expires_at"`
			PinRequired bool   `json:"pin_required"`
			PIN         string `json:"pin"`
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("server returned %s: %s", resp.Status, string(raw))
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return err
		}
		fmt.Printf("URL: %s\nExpires: %s\n", result.URL, time.Unix(result.ExpiresAt, 0).Format("2006-01-02 15:04"))
		if result.PinRequired {
			fmt.Printf("PIN: %s\n", result.PIN)
		}
		return nil
	},
}

func init() {
	shareCmd.Flags().StringVar(&shareFrom, "from", "", "server address host:port")
	shareCmd.Flags().StringVar(&shareToken, "token", "", "server token (or LANBOX_TOKEN)")
	shareCmd.Flags().StringVar(&sharePin, "pin", "", "server PIN (or LANBOX_PIN)")
	shareCmd.Flags().StringVar(&shareExpires, "expires", "30m", "share lifetime (max 24h)")
	shareCmd.Flags().BoolVar(&shareNeedPIN, "pin-require", false, "protect the share with a PIN")
	shareCmd.Flags().BoolVar(&shareUpload, "upload", false, "share a directory as an upload inbox")
	rootCmd.AddCommand(shareCmd)
}
