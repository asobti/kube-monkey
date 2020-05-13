package param

const (
	// DryRun logs but does not terminate pods
	// Type: bool
	// Default: true
	DryRun = "kubemonkey.dry_run"

	// HarmMode specifies how monkey do harmful work in the cluster
	// Type: string
	// Default: delete_pod
	// Available values: delete_pod, exec_pod
	// delete_pod: monkey will kill pod in cluster
	// exec_pod: monkey will exec command specified in harm_value in pod
	HarmType = "kubemonkey.harm_type"

	// Timezone specifies the timezone to use when
	// scheduling Pod terminations
	// Type: string
	// Default: America/Los_Angeles
	Timezone = "kubemonkey.time_zone"

	// RunHour specifies the hour of the weekday
	// when the scheduler should run to schedule terminations
	// Must be less than StartHour, and [0,23]
	// Type: int
	// Default: 8
	RunHour = "kubemonkey.run_hour"

	// StartHour specifies the hour beginning at
	// which pod terminations may occur
	// Should be set to a time when service owners are expected
	// to be available
	// Must be less than EndHour, and [0, 23]
	// Type: int
	// Default: 10
	StartHour = "kubemonkey.start_hour"

	// EndHour specifies the end hour beyond which no pod
	// terminations will occur
	// Should be set to a time when service owners are
	// expected to be available
	// Must be [0,23]
	// Type: int
	// Default: 16
	EndHour = "kubemonkey.end_hour"

	// GracePeriodSec specifies the amount of time in
	// seconds a pod is given to shut down gracefully,
	// before Kubernetes does a hard kill
	// Type: int
	// Default: 5
	GracePeriodSec = "kubemonkey.graceperiod_sec"

	// WhitelistedNamespaces specifies a list of
	// namespaces where terminations are valid
	// Default is defined by metav1.NamespaceDefault
	// To allow all namespaces use [""]
	// Type: list
	// Default: [ "default" ]
	WhitelistedNamespaces = "kubemonkey.whitelisted_namespaces"

	// BlacklistedNamespaces specifiesd a list of namespaces
	// for which terminations should never
	// be carried out.
	// Default is defined by metav1.NamespaceSystem
	// To block no namespaces use [""]
	// Type: list
	// Default: [ "kube-system" ]
	BlacklistedNamespaces = "kubemonkey.blacklisted_namespaces"

	// ClusterAPIServerHost specifies the host URL for Kubernetes
	// cluster APIServer. Use this config if the apiserver IP
	// address provided by in-cluster config
	// does not work for you because your certificate does not
	// conatain the right SAN
	// Type: string
	// Default: No default. If not specified, URL provided
	// by in-cluster config is used
	ClusterAPIServerHost = "kubernetes.host"

	// DebugEnabled enables debug mode
	// Type: bool
	// Default: false
	DebugEnabled = "debug.enabled"

	// DebugScheduleDelay delays duration
	// in sec after kube-monkey is launched
	// after which scheduling is run
	// Use when debugging to run scheduling sooner
	// Type: int
	// Default: 30
	DebugScheduleDelay = "debug.schedule_delay"

	// DebugForceShouldKill guarantees terminations
	// to be scheduled for all eligible Deployments,
	// i.e., probability of kill = 1
	// Type: bool
	// Default: false
	DebugForceShouldKill = "debug.force_should_kill"

	// DebugScheduleImmediateKill schedules pod terminations
	// sometime in the next 60 sec to facilitate
	// debugging (instead of the hours specified by
	// StartHour and EndHour)
	// Type: bool
	// Default: false
	DebugScheduleImmediateKill = "debug.schedule_immediate_kill"
)
