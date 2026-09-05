package captions

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucasepe/x/filecache"
)

var errEmojiAssetNotFound = errors.New("emoji asset not found")

// defaultEmojiCacheDir returns the emoji cache directory from the environment or the OS cache root.
func defaultEmojiCacheDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(emojiCacheEnv)); dir != "" {
		return dir, nil
	}
	return filecache.NewCacheDir(filepath.Join("cliphub", "emoji"))
}

// loadEmojiImage loads a Twemoji PNG from the file-backed cache, downloading it on first use.
func loadEmojiImage(cluster, cacheDir string) (image.Image, error) {
	cache, err := filecache.New(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("open emoji cache: %w", err)
	}

	key := emojiURL(cluster)
	data, err := cache.Get(key)
	if errors.Is(err, filecache.ErrNotFound) {
		data, err = downloadEmoji(cluster)
		if err != nil {
			return nil, err
		}
		if err := cache.Put(key, data); err != nil {
			return nil, fmt.Errorf("write emoji cache: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("read emoji cache: %w", err)
	}

	return decodeEmoji(data, key)
}

// decodeEmoji decodes PNG bytes into an image.
func decodeEmoji(data []byte, label string) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode emoji asset %q: %w", label, err)
	}

	return img, nil
}

// emojiFilename converts a grapheme cluster into the lowercase codepoint filename used by Twemoji.
func emojiFilename(cluster string) string {
	var codepoints []string
	for _, r := range cluster {
		if r == 0xFE0F {
			continue
		}
		codepoints = append(codepoints, fmt.Sprintf("%x", r))
	}

	return strings.Join(codepoints, "-") + ".png"
}

// emojiURL returns the Twemoji CDN URL for a grapheme cluster.
func emojiURL(cluster string) string {
	return twemojiBaseURL + "/" + emojiFilename(cluster)
}

// downloadEmoji fetches one Twemoji PNG.
func downloadEmoji(cluster string) ([]byte, error) {
	url := emojiURL(cluster)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download emoji %q: %w", cluster, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %q", errEmojiAssetNotFound, cluster)
		}
		return nil, fmt.Errorf("download emoji %q: %s", cluster, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read emoji %q: %w", cluster, err)
	}

	return data, nil
}
