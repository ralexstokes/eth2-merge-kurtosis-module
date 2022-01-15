package cl

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/log_levels"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type CLClientLauncher interface {
	// Launches both a Beacon client AND a validator
	Launch(
		enclaveCtx *enclaves.EnclaveContext,
		serviceId services.ServiceID,
		logLevel log_levels.ParticipantLogLevel,
		// If nil, the node will be launched as a bootnode
		bootnodeContext *CLClientContext,
		elClientContext *el.ELClientContext,
		nodeKeystoreDirpaths *cl.NodeTypeKeystoreDirpaths,
	) (
		resultClientCtx *CLClientContext,
		resultErr error,
	)
}
