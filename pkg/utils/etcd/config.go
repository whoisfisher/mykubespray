package etcd

import (
	"fmt"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"os"
	"strings"
	"sync"
)

type Config struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewConfig() *Config {
	return &Config{
		data: make(map[string]string),
	}
}

func (c *Config) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.data[key]
	return value, ok
}

func SetEnvVars(content string) error {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			err := os.Setenv(parts[0], parts[1])
			if err != nil {
				logger.GetLogger().Errorf("Cannot set enviroment %s: %v", parts[0], err)
				return fmt.Errorf("Cannot set enviroment %s: %v", parts[0], err)
			}
		}
	}

	return nil
}
