package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	receiveFrom   string
	receiveOut    string
	receiveToken  string
	receiveResume bool
	receivePin    string
)

var receiveCmd = &cobra.Command{
	Use:   "receive <path>",
	Short: "Download a file from a running server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remote := args[0]
		if receiveFrom == "" {
			return fmt.Errorf("missing --from (e.g. --from 192.168.1.10:8080)")
		}
		token := receiveToken
		if token == "" {
			token = os.Getenv("LANBOX_TOKEN")
		}
		out := receiveOut
		if out == "" || isDir(out) {
			out = filepath.Join(out, filepath.Base(remote))
		}
		var offset int64
		if receiveResume {
			if info, err := os.Stat(out); err == nil && !info.IsDir() {
				offset = info.Size()
			}
		}
		url := apiURL(receiveFrom, "/api/v1/files/download?path="+remote)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if pin := pinFor(receivePin); pin != "" {
			req.Header.Set("X-PIN", pin)
		}
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
			fmt.Printf("Resuming from %s...\n", humanBytes(offset))
		}
		start := time.Now()
		resp, err := tlsClient.Do(req)
		if err != nil {
			return fmt.Errorf("cannot connect — is lanbox serve running? %w", err)
		}
		defer resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			offset = 0 // server ignored Range or fresh start
		case http.StatusPartialContent:
		default:
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
			return fmt.Errorf("server returned %s: %s", resp.Status, string(body))
		}
		var f *os.File
		if offset > 0 {
			f, err = os.OpenFile(out, os.O_WRONLY|os.O_APPEND, 0o644)
		} else {
			f, err = os.Create(out)
		}
		if err != nil {
			return err
		}
		n, err := io.Copy(f, resp.Body)
		_ = f.Close()
		if err != nil {
			return err
		}
		el := time.Since(start).Seconds()
		fmt.Printf("Received: %s (%.1f MB) in %.1fs\n", out, float64(offset+n)/1024/1024, el)
		return verifyDownload(receiveFrom, token, remote, out)
	},
}

// verifyDownload compares the local file hash against the server checksum.
func verifyDownload(host, token, remote, local string) error {
	url := apiURL(host, "/api/v1/files/checksum?path="+remote)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	setAuth(req, token, pinFor(receivePin))
	resp, err := tlsClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result struct {
		SHA256 string `json:"sha256"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	localSum, err := hashFile(local)
	if err != nil {
		return err
	}
	if result.SHA256 != "" && result.SHA256 == localSum {
		fmt.Printf("Checksum: sha256:%s\nVerified\n", localSum)
		return nil
	}
	return fmt.Errorf("checksum mismatch: local sha256:%s server sha256:%s", localSum, result.SHA256)
}

func isDir(p string) bool {
	if p == "" {
		return true
	}
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func init() {
	receiveCmd.Flags().StringVar(&receiveFrom, "from", "", "server address host:port")
	receiveCmd.Flags().StringVar(&receiveOut, "out", "", "output file or directory")
	receiveCmd.Flags().StringVar(&receiveToken, "token", "", "server token (or LANBOX_TOKEN)")
	receiveCmd.Flags().BoolVar(&receiveResume, "resume", false, "resume from existing partial file")
	receiveCmd.Flags().StringVar(&receivePin, "pin", "", "server PIN (or LANBOX_PIN)")
	rootCmd.AddCommand(receiveCmd)
}
