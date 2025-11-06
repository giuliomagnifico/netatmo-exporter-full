package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var (
	thermostatLabels = []string{"home_id", "home_name", "room_id", "room_name"}

	thermostatTemperatureDesc = prometheus.NewDesc(
		prefix+"thermostat_temperature",
		"Netatmo Energy measured room temperature in degrees Celsius.",
		thermostatLabels,
		nil,
	)

	thermostatSetpointDesc = prometheus.NewDesc(
		prefix+"thermostat_setpoint",
		"Netatmo Energy target setpoint temperature in degrees Celsius.",
		thermostatLabels,
		nil,
	)

	thermostatBoilerStatusDesc = prometheus.NewDesc(
		prefix+"thermostat_boiler_status",
		"Netatmo Energy boiler status (1=on, 0=off). Per-room when possibile, otherwise per-home.",
		thermostatLabels,
		nil,
	)
)


type ThermostatCollector struct {
	log       logrus.FieldLogger
	tokenFunc func() (*oauth2.Token, error)
}

func NewThermostatCollector(log logrus.FieldLogger, tokenFunc func() (*oauth2.Token, error)) *ThermostatCollector {
	return &ThermostatCollector{
		log:       log,
		tokenFunc: tokenFunc,
	}
}

func (c *ThermostatCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- thermostatTemperatureDesc
	ch <- thermostatSetpointDesc
	ch <- thermostatBoilerStatusDesc
}

// Collect implementa prometheus.Collector.
func (c *ThermostatCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	token, err := c.tokenFunc()
	if err != nil {
		c.log.Errorf("ThermostatCollector: error getting token: %v", err)
		return
	}
	if token == nil || !token.Valid() {
		c.log.Debug("ThermostatCollector: token not available or invalid, skipping collection.")
		return
	}

	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	homes, err := fetchHomes(ctx, httpClient)
	if err != nil {
		c.log.Errorf("ThermostatCollector: error fetching homesdata: %v", err)
		return
	}

	for _, home := range homes.Body.Homes {
		status, err := fetchHomeStatus(ctx, httpClient, home.ID)
		if err != nil {
			c.log.Errorf("ThermostatCollector: error fetching homestatus for %s: %v", home.ID, err)
			continue
		}

		h := status.Body.Home

		homeID := h.ID
		if homeID == "" {
			homeID = home.ID
		}

		homeName := h.Name
		if homeName == "" {
			homeName = home.Name
		}

		boilerByRoom := map[string]float64{}
		var homeBoiler *float64

		for _, mod := range h.Modules {
			if mod.BoilerStatus == nil {
				continue
			}

			v := 0.0
			if *mod.BoilerStatus {
				v = 1.0
			}

			if mod.RoomID != "" {
				boilerByRoom[mod.RoomID] = v
			}

			if homeBoiler == nil {
				tmp := v
				homeBoiler = &tmp
			} else if v > *homeBoiler {
				*homeBoiler = v
			}
		}

		for _, room := range h.Rooms {
			labels := []string{homeID, homeName, room.ID, room.Name}

			if room.MeasuredTemperature != nil {
				ch <- prometheus.MustNewConstMetric(
					thermostatTemperatureDesc,
					prometheus.GaugeValue,
					*room.MeasuredTemperature,
					labels...,
				)
			}

			if room.SetpointTemperature != nil {
				ch <- prometheus.MustNewConstMetric(
					thermostatSetpointDesc,
					prometheus.GaugeValue,
					*room.SetpointTemperature,
					labels...,
				)
			}

			if val, ok := boilerByRoom[room.ID]; ok {
				ch <- prometheus.MustNewConstMetric(
					thermostatBoilerStatusDesc,
					prometheus.GaugeValue,
					val,
					labels...,
				)
			}
		}

		if homeBoiler != nil {
			labels := []string{homeID, homeName, "", ""}
			ch <- prometheus.MustNewConstMetric(
				thermostatBoilerStatusDesc,
				prometheus.GaugeValue,
				*homeBoiler,
				labels...,
			)
		}
	}
}

type homesDataResponse struct {
	Body struct {
		Homes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"homes"`
	} `json:"body"`
}

type homeStatusResponse struct {
	Body struct {
		Home struct {
			ID      string        `json:"id"`
			Name    string        `json:"name"`
			Rooms   []roomStatus  `json:"rooms"`
			Modules []moduleStatus `json:"modules"`
		} `json:"home"`
	} `json:"body"`
}

type roomStatus struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	MeasuredTemperature *float64 `json:"therm_measured_temperature"`
	SetpointTemperature *float64 `json:"therm_setpoint_temperature"`
}

type moduleStatus struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	RoomID       string `json:"room_id"`
	BoilerStatus *bool  `json:"boiler_status,omitempty"`
}

func fetchHomes(ctx context.Context, client *http.Client) (*homesDataResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.netatmo.com/api/homesdata", nil)
	if err != nil {
		return nil, fmt.Errorf("creating homesdata request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing homesdata request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("homesdata request failed: status %s", resp.Status)
	}

	var result homesDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding homesdata response: %w", err)
	}

	return &result, nil
}

func fetchHomeStatus(ctx context.Context, client *http.Client, homeID string) (*homeStatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.netatmo.com/api/homestatus", nil)
	if err != nil {
		return nil, fmt.Errorf("creating homestatus request: %w", err)
	}

	q := req.URL.Query()
	q.Set("home_id", homeID)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing homestatus request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("homestatus request failed: status %s", resp.Status)
	}

	var result homeStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding homestatus response: %w", err)
	}

	return &result, nil
}
