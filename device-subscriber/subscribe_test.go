/*
 * Copyright (c) 2022. Hiroki Okui
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfig_Validate(t *testing.T) {

	newValidStruct := func(t func(*Config)) *Config {
		cfg := &Config{
			Device:        "device1",
			Addr:          ":9339",
			AggregatorURL: "http://localhost:8000",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *Config)
		wantErr   bool
	}{
		{
			"ok",
			func(cfg *Config) {},
			false,
		},
		{
			"bad: device is empty",
			func(cfg *Config) {
				cfg.Device = ""
			},
			true,
		},
		{
			"bad: addr is empty",
			func(cfg *Config) {
				cfg.Addr = ""
			},
			true,
		},
		{
			"bad: aggregator-url is empty",
			func(cfg *Config) {
				cfg.AggregatorURL = ""
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidStruct(tt.transform)
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestPostDeviceConfig(t *testing.T) {
	deviceConfig := "dummy"

	t.Run("ok", func(t *testing.T) {
		cfg := Config{
			Device: "device1",
		}
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req SaveConfigRequest
			exitOnErr(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, req.Device, cfg.Device)
			assert.Equal(t, req.Config, deviceConfig)
		}))
		cfg.AggregatorURL = s.URL

		err := PostDeviceConfig(cfg, []byte(deviceConfig))
		assert.Nil(t, err)
	})

	t.Run("bad: error response", func(t *testing.T) {
		cfg := Config{
			Device: "device1",
		}
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}))
		cfg.AggregatorURL = s.URL

		err := PostDeviceConfig(cfg, []byte(deviceConfig))
		assert.Error(t, err)
	})

	t.Run("bad: wrong url", func(t *testing.T) {
		cfg := Config{
			Device: "device1",
		}
		cfg.AggregatorURL = ":60000"

		err := PostDeviceConfig(cfg, []byte(deviceConfig))
		assert.Error(t, err)
	})

	t.Run("bad: connection error", func(t *testing.T) {
		cfg := Config{
			Device: "device1",
		}
		cfg.AggregatorURL = "http://localhost:60000"

		err := PostDeviceConfig(cfg, []byte(deviceConfig))
		assert.Error(t, err)
	})
}
