package config

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Config struct {
	DataDir   string
	ScanRoot  string
	ScanDepth int
	ScanSkip  []string
}

func Defaults() Config {
	home, _ := os.UserHomeDir()
	skip := []string{"/dev", "/System/Volumes", "/net", "/home"}
	if runtime.GOOS == "linux" {
		skip = []string{"/proc", "/sys", "/dev", "/run"}
	}
	return Config{
		DataDir:   filepath.Join(home, ".dwatch"),
		ScanRoot:  "/",
		ScanDepth: 5,
		ScanSkip:  skip,
	}
}

// Load reads a config file, returning Defaults() if the file does not exist.
func Load(path string) (Config, error) {
	cfg := Defaults()

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)

		switch k {
		case "data_dir":
			cfg.DataDir = expandHome(v)
		case "scan_root":
			cfg.ScanRoot = v
		case "scan_depth":
			if n, err := strconv.Atoi(v); err == nil {
				cfg.ScanDepth = n
			}
		case "scan_skip":
			parts := strings.Split(v, ",")
			cfg.ScanSkip = cfg.ScanSkip[:0]
			for _, p := range parts {
				if p = strings.TrimSpace(p); p != "" {
					cfg.ScanSkip = append(cfg.ScanSkip, p)
				}
			}
		}
	}
	return cfg, s.Err()
}

func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
