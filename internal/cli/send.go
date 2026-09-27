package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/lanbox/internal/transfer"
)

var (
	sendTo    string
	sendToken string
)

var sendCmd = &cobra.Command{
	Use:   "send <file>",
	Short: "Upload a file to a running server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		local := args[0]
		f, err := os.Open(local)
		if err != nil {
			return fmt.Errorf("cannot open %s: %w", local, err)
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			return fmt.Errorf("not a file: %s", local)
		}
		if sendTo == "" {
			return fmt.Errorf("missing --to (e.g. --to 192.168.1.10:8080)")
		}
		token := sendToken
		if token == "" {
			token = os.Getenv("LANBOX_TOKEN")
		}

		pr, pw := io.Pipe()
		w := multipart.NewWriter(pw)
		go func() {
			part, err := w.CreateFormFile("file", filepath.Base(local))
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			_, err = io.Copy(part, f)
			_ = w.Close()
			_ = pw.CloseWithError(err)
		}()

		url := fmt.Sprintf("http://%s/api/v1/files/upload?path=/", sendTo)
		req, err := http.NewRequest("POST", url, pr)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", w.FormDataContentType())
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		start := time.Now()
		prog := &transfer.Progress{Total: info.Size()}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("cannot connect — is lanbox serve running? %w", err)
		}
		defer resp.Body.Close()
		var result struct {
			Name     string `json:"name"`
			Size     int64  `json:"size"`
			Checksum string `json:"checksum"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&result)
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("server returned %s", resp.Status)
		}
		prog.Done = info.Size()
		el := time.Since(start).Seconds()
		speed := float64(info.Size()) / el / 1024 / 1024
		fmt.Printf("%s\n%s\nTransferred: %s\nTime: %.1fs\nSpeed: %.1f MB/s\n",
			filepath.Base(local), prog.String(), humanBytes(info.Size()), el, speed)
		localSum, err := hashFile(local)
		if err != nil {
			return err
		}
		if result.Checksum != "" && result.Checksum == "sha256:"+localSum {
			fmt.Printf("Checksum: sha256:%s\nVerified\n", localSum)
		} else {
			return fmt.Errorf("checksum mismatch: local sha256:%s server %s", localSum, result.Checksum)
		}
		return nil
	},
}

func humanBytes(n int64) string {
	const u = 1024
	if n < u {
		return fmt.Sprintf("%d B", n)
	}
	v := float64(n)
	for _, s := range []string{"KB", "MB", "GB"} {
		v /= u
		if v < u || s == "GB" {
			return fmt.Sprintf("%.1f %s", v, s)
		}
	}
	return fmt.Sprintf("%.1f GB", v)
}

// hashFile streams a local file through SHA-256.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := transfer.NewHasher()
	if _, err := io.Copy(io.Discard, h.Reader(f)); err != nil {
		return "", err
	}
	return h.Sum(), nil
}

func init() {
	sendCmd.Flags().StringVar(&sendTo, "to", "", "server address host:port")
	sendCmd.Flags().StringVar(&sendToken, "token", "", "server token (or LANBOX_TOKEN)")
	rootCmd.AddCommand(sendCmd)
}
