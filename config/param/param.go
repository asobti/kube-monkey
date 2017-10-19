package param

const (
	// If set to true, Pod terminations are only logged and pods are
	// not actually killed.
	// Type: bool
	// Default: true
	DryRun = "kubemonkey.dry_run"

	// If set to true, the deployments selection process is performed considering the
	// kube-monkey/enabled flag. If it is false, are selected all deployments that are
	// not labeled with kube-monkey/disabled.
	// Type: bool
	// Default: true
	SafeMode = "kubemonkey.safe_mode"

	// When you create a K8S deployment, you should have a label defining the owned pods.
	// You can set the name of that label with this parameter in order to match your cluster settings.
	// Type: string
	// Default: kube-monkey/identifier
	IdentLabelKey = "kubemonkey.ident_label_key"

	// The timezone to use when scheduling Pod terminations
	// Type: string
	// Default: America/Los_Angeles
	Timezone = "kubemonkey.time_zone"

	// The hour of the weekday when the scheduler should run
	// to schedule terminations
	// Must be less than StartHour, and [0,23]
	// Type: int
	// Default: 8
	RunHour = "kubemonkey.run_hour"

	// The hour beginning at which pod terminations may occur
	// Should be set to a time when service owners are expected
	// to be available
	// Must be less than EndHour, and [0, 23]
	// Type: int
	// Default: 10
	StartHour = "kubemonkey.start_hour"

	// The end hour beyond which no pod terminations will occur
	// Should be set to a time when service owners are expected
	// to be available
	// Must be [0,23]
	// Type: int
	// Default: 16
	EndHour = "kubemonkey.end_hour"

	// The amount of time in seconds a pod is given
	// to shut down gracefully, before Kubernetes does
	// a hard kill
	// Type: int
	// Default: 5
	GracePeriodSec = "kubemonkey.graceperiod_sec"

	// A list of namespaces for which terminations should never
	// be carried out.
	// Type: list
	// Default: [ "kube-system" ]
	BlacklistedNamespaces = "kubemonkey.blacklisted_namespaces"

	// Host URL for Kubernetes cluster APIServer. Use this config
	// if the apiserver IP address provided by in-cluster config
	// does not work for you because your certificate does not
	// conatain the right SAN
	// Type: string
	// Default: No default. If not specified, URL provided
	// by in-cluster config is used
	ClusterAPIServerHost = "kubernetes.host"

	// Set to true to enable debug mode
	// Type: bool
	// Default: false
	DebugEnabled = "debug.enabled"

	// Delay duration in sec after kube-monkey is launched
	// after which scheduling is run
	// Use when debugging to run scheduling sooner
	// Type: int
	// Default: 30
	DebugScheduleDelay = "debug.schedule_delay"

	// If set to true, terminations will be guaranteed
	// to be scheduled for all eligible Deployments,
	// i.e., probability of kill = 1
	// Type: bool
	// Default: false
	DebugForceShouldKill = "debug.force_should_kill"

	// If set to true, pod terminations will be scheduled
	// sometime in the next 60 sec to facilitate
	// debugging (instead of the hours specified by
	// StartHour and EndHour)
	// Type: bool
	// Default: false
	DebugScheduleImmediateKill = "debug.schedule_immediate_kill"
)
