package ice

import "errors"

var (
	// ErrUnknownType indicates an error with Unknown info.
	ErrUnknownType = errors.New("Unknown")

	// ErrSchemeType indicates the scheme type could not be parsed.
	ErrSchemeType = errors.New("unknown scheme type")

	// ErrSTUNQuery indicates query arguments are provided in a STUN URL.
	ErrSTUNQuery = errors.New("queries not supported in stun address")

	// ErrInvalidQuery indicates an malformed query is provided.
	ErrInvalidQuery = errors.New("invalid query")

	// ErrHost indicates malformed hostname is provided.
	ErrHost = errors.New("invalid hostname")

	// ErrPort indicates malformed port is provided.
	ErrPort = errors.New("invalid port")

	// ErrProtoType indicates an unsupported transport type was provided.
	ErrProtoType = errors.New("invalid transport protocol type")

	// ErrClosed indicates the agent is closed
	ErrClosed = errors.New("the agent is closed")

	// ErrNoCandidatePairs indicates agent does not have a valid candidate pair
	ErrNoCandidatePairs = errors.New("no candidate pairs available")

	// ErrCanceledByCaller indicates agent connection was canceled by the caller
	ErrCanceledByCaller = errors.New("connecting canceled by caller")

	// ErrMultipleStart indicates agent was started twice
	ErrMultipleStart = errors.New("attempted to start agent twice")

	// ErrRemoteUfragEmpty indicates agent was started with an empty remote ufrag
	ErrRemoteUfragEmpty = errors.New("remote ufrag is empty")

	// ErrRemotePwdEmpty indicates agent was started with an empty remote pwd
	ErrRemotePwdEmpty = errors.New("remote pwd is empty")

	// ErrNoOnCandidateHandler indicates agent was started without OnCandidate
	// while running in trickle mode.
	ErrNoOnCandidateHandler = errors.New("no OnCandidate provided")

	// ErrMultipleGatherAttempted indicates GatherCandidates has been called multiple times
	ErrMultipleGatherAttempted = errors.New("attempting to gather candidates during gathering state")

	// ErrUsernameEmpty indicates agent was give TURN URL with an empty Username
	ErrUsernameEmpty = errors.New("username is empty")

	// ErrPasswordEmpty indicates agent was give TURN URL with an empty Password
	ErrPasswordEmpty = errors.New("password is empty")

	// ErrAddressParseFailed indicates we were unable to parse a candidate address
	ErrAddressParseFailed = errors.New("failed to parse address")
)
