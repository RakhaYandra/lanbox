package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	receiveFrom  string
	receiveOut   string
	receiveToken string
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
		url := fmt.Sprintf("http://%s/api/v1/files/download?path=%s", receiveFrom, remote)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		start := time.Now()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("cannot connect — is lanbox serve running? %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
			return fmt.Errorf("server returned %s: %s", resp.Status, string(body))
		}
		out := receiveOut
		if out == "" || isDir(out) {
			out = filepath.Join(out, filepath.Base(remote))
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		n, err := io.Copy(f, resp.Body)
		_ = f.Close()
		if err != nil {
			_ = os.Remove(out)
			return err
		}
		el := time.Since(start).Seconds()
		fmt.Printf("Received: %s (%.1f MB) in %.1fs\n", out, float64(n)/1024/1024, el)
		return nil
	},
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
	rootCmd.AddCommand(receiveCmd)
}
