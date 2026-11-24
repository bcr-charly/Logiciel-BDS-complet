package kernel

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Conf est la configuration clé=valeur lue dans conf.txt (redémarrage requis
// pour appliquer, conformément à la spec).
type Conf struct {
	values map[string]string
	path   string
}

func LoadConf(path string) (*Conf, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	c := &Conf{values: map[string]string{}, path: path}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		c.values[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return c, sc.Err()
}

func (c *Conf) Get(key, def string) string {
	if v, ok := c.values[key]; ok && v != "" {
		return v
	}
	return def
}

func (c *Conf) Has(key string) bool {
	v, ok := c.values[key]
	return ok && v != ""
}

func (c *Conf) Int(key string, def int) int {
	if v, ok := c.values[key]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func (c *Conf) Path() string { return c.path }

// SetValue ajoute/remplace une valeur en mémoire (ex. override par flag).
func (c *Conf) SetValue(key, value string) { c.values[key] = value }
