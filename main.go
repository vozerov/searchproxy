package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Proxies []ProxyEntry `yaml:"proxies"`
	Static  StaticConfig `yaml:"static"`
	Health  HealthConfig `yaml:"health"`
}

type ProxyEntry struct {
	Listen string `yaml:"listen"`
	Target string `yaml:"target"`
}

type StaticConfig struct {
	Listen string `yaml:"listen"`
	Dir    string `yaml:"dir"`
}

type HealthConfig struct {
	Listen string `yaml:"listen"`
}

var pending atomic.Int32

func proxy(listen, target string) {
	l, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("tcp %s -> %s", listen, target)
	pending.Add(-1)
	for {
		c, err := l.Accept()
		if err != nil {
			continue
		}
		go func() {
			defer c.Close()
			// net.Dial reads /etc/resolv.conf and honors the search list.
			b, err := net.Dial("tcp", target)
			if err != nil {
				log.Printf("dial %s: %v", target, err)
				return
			}
			defer b.Close()
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); io.Copy(b, c) }()
			go func() { defer wg.Done(); io.Copy(c, b) }()
			wg.Wait()
		}()
	}
}

func health(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if pending.Load() == 0 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok\n"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready\n"))
	})
	log.Printf("health %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func main() {
	cfgPath := flag.String("config", "/etc/searchproxy/config.yaml", "path to config file")
	flag.Parse()

	data, err := os.ReadFile(*cfgPath)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("parse config: %v", err)
	}

	pending.Store(int32(len(cfg.Proxies)))

	for _, p := range cfg.Proxies {
		go proxy(p.Listen, p.Target)
	}

	healthAddr := cfg.Health.Listen
	if healthAddr == "" {
		healthAddr = ":8081"
	}
	go health(healthAddr)

	if cfg.Static.Dir != "" {
		log.Printf("static %s -> %s", cfg.Static.Listen, cfg.Static.Dir)
		log.Fatal(http.ListenAndServe(cfg.Static.Listen,
			http.FileServer(http.Dir(cfg.Static.Dir))))
	}
	select {}
}
