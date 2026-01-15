package metrics

import "github.com/prometheus/client_golang/prometheus"

// metrics variables
var (
	SendCommandCalls = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "command_service_send_comand_total",
			Help: "Total number of /send_command calls",
		},
	)

	SendCommandHistogramm = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "command_service_send_command_duration_seconds",
			Help:    "Duration of SendCommand query",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0},
		},
	)

	CommadsPollCalls = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "command_service_commands_poll_total",
			Help: "Total number of /commands/poll calls",
		},
	)

	CommandsPollHistogramm = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "command_service_comands_poll_duration_seconds",
			Help:    "Duration of PollCommands query",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0},
		},
	)

	CommandsAckCalls = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "command_service_commands_ack_total",
			Help: "Total number of /commands/ack calls",
		},
	)

	CommandsAckHistogramm = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "command_service_commands_ack_duration_seconds",
			Help:    "Duration of AckCommands query",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0},
		},
	)
)

func init() {
	prometheus.MustRegister(SendCommandCalls)
	prometheus.MustRegister(SendCommandHistogramm)

	prometheus.MustRegister(CommadsPollCalls)
	prometheus.MustRegister(CommandsPollHistogramm)

	prometheus.MustRegister(CommandsAckCalls)
	prometheus.MustRegister(CommandsAckHistogramm)
}
