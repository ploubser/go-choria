package network

import (
	"context"
	"strconv"
	"strings"
	"time"

	gnatsd "github.com/nats-io/nats-server/v2/server"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	connectionsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_connections",
		Help: "Current connections on the network broker",
	}, []string{"identity"})

	totalConnectionsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_total_connections",
		Help: "Total connections received since start",
	}, []string{"identity"})

	routesGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_routes",
		Help: "Current active routes to other brokers",
	}, []string{"identity"})

	remotesGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_remotes",
		Help: "Current active connections to other brokers",
	}, []string{"identity"})

	leafsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_leafnode_remotes",
		Help: "Current active connections to other leafnodes",
	}, []string{"identity"})

	inMsgsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_in_msgs",
		Help: "Messages received by the network broker",
	}, []string{"identity"})

	outMsgsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_out_msgs",
		Help: "Messages sent by the network broker",
	}, []string{"identity"})

	inBytesGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_in_bytes",
		Help: "Total size of messages received by the network broker",
	}, []string{"identity"})

	outBytesGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_out_bytes",
		Help: "Total size of messages sent by the network broker",
	}, []string{"identity"})

	slowConsumerGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_slow_consumers",
		Help: "Total number of clients who were considered slow consumers",
	}, []string{"identity"})

	subscriptionsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_subscriptions",
		Help: "Number of active subscriptions to subjects on this broker",
	}, []string{"identity"})

	leafTTGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_leafnode_rtt_ms",
		Help: "RTT for the Leafnode connection",
	}, []string{"identity", "host", "port", "account"})

	leafMsgsInGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_leafnode_in_msgs",
		Help: "Messages received over the leafnode connection",
	}, []string{"identity", "host", "port", "account"})

	leafMsgsOutGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_leafnode_out_msgs",
		Help: "Messages sent over the leafnode connection",
	}, []string{"identity", "host", "port", "account"})

	leafBytesInGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_leafnode_in_bytes",
		Help: "Bytes received over the leafnode connection",
	}, []string{"identity", "host", "port", "account"})

	leafBytesOutGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_leafnode_out_bytes",
		Help: "Total size of messages sent over the leafnode connection",
	}, []string{"identity", "host", "port", "account"})

	leafSubscriptionsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "choria_network_leafnode_subscriptions",
		Help: "Number of active subscriptions to subjects on this leafnode",
	}, []string{"identity", "host", "port", "account"})
)

func init() {
	prometheus.MustRegister(connectionsGauge)
	prometheus.MustRegister(totalConnectionsGauge)
	prometheus.MustRegister(routesGauge)
	prometheus.MustRegister(remotesGauge)
	prometheus.MustRegister(leafsGauge)
	prometheus.MustRegister(inMsgsGauge)
	prometheus.MustRegister(outMsgsGauge)
	prometheus.MustRegister(inBytesGauge)
	prometheus.MustRegister(outBytesGauge)
	prometheus.MustRegister(slowConsumerGauge)
	prometheus.MustRegister(subscriptionsGauge)
	prometheus.MustRegister(leafTTGauge)
	prometheus.MustRegister(leafMsgsInGauge)
	prometheus.MustRegister(leafMsgsOutGauge)
	prometheus.MustRegister(leafBytesInGauge)
	prometheus.MustRegister(leafBytesOutGauge)
	prometheus.MustRegister(leafSubscriptionsGauge)
}

func (s *Server) getVarz() (*gnatsd.Varz, error) {
	return s.gnatsd.Varz(&gnatsd.VarzOptions{})
}

func (s *Server) getLeafz() (*gnatsd.Leafz, error) {
	return s.gnatsd.Leafz(&gnatsd.LeafzOptions{Subscriptions: false})
}

func (s *Server) publishStats(ctx context.Context, interval time.Duration) {
	if s.opts.HTTPPort == 0 {
		return
	}

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ticker.C:
			log.Debug("Starting NATS /varz update")

			s.updatePrometheus()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) updatePrometheus() {
	varz, err := s.getVarz()
	if err != nil {
		log.Errorf("Could not publish network broker stats: %s", err)
		return
	}

	i := s.config.Identity

	connectionsGauge.WithLabelValues(i).Set(float64(varz.Connections))
	totalConnectionsGauge.WithLabelValues(i).Set(float64(varz.TotalConnections))
	routesGauge.WithLabelValues(i).Set(float64(varz.Routes))
	remotesGauge.WithLabelValues(i).Set(float64(varz.Remotes))
	inMsgsGauge.WithLabelValues(i).Set(float64(varz.InMsgs))
	outMsgsGauge.WithLabelValues(i).Set(float64(varz.OutMsgs))
	inBytesGauge.WithLabelValues(i).Set(float64(varz.InBytes))
	outBytesGauge.WithLabelValues(i).Set(float64(varz.OutBytes))
	slowConsumerGauge.WithLabelValues(i).Set(float64(varz.SlowConsumers))
	subscriptionsGauge.WithLabelValues(i).Set(float64(varz.Subscriptions))
	leafsGauge.WithLabelValues(i).Set(float64(varz.Leafs))

	leafz, err := s.getLeafz()
	if err != nil {
		log.Errorf("Could not publish network broker stats: %s", err)
		return
	}

	for _, leaf := range leafz.Leafs {
		leafMsgsInGauge.WithLabelValues(i, leaf.IP, strconv.Itoa(leaf.Port), leaf.Account).Set(float64(leaf.InMsgs))
		leafMsgsOutGauge.WithLabelValues(i, leaf.IP, strconv.Itoa(leaf.Port), leaf.Account).Set(float64(leaf.OutMsgs))
		leafBytesInGauge.WithLabelValues(i, leaf.IP, strconv.Itoa(leaf.Port), leaf.Account).Set(float64(leaf.InBytes))
		leafBytesOutGauge.WithLabelValues(i, leaf.IP, strconv.Itoa(leaf.Port), leaf.Account).Set(float64(leaf.OutBytes))
		leafSubscriptionsGauge.WithLabelValues(i, leaf.IP, strconv.Itoa(leaf.Port), leaf.Account).Set(float64(leaf.NumSubs))

		rtt, err := strconv.Atoi(strings.TrimSuffix(leaf.RTT, "ms"))
		if err == nil {
			leafTTGauge.WithLabelValues(i, leaf.IP, strconv.Itoa(leaf.Port), leaf.Account).Set(float64(rtt))
		}
	}
}
