package participant_network
import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/log_levels"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

const (
	clClientServiceIdPrefix = "cl-client-"
	elClientServiceIdPrefix = "el-client-"

	bootParticipantIndex = 0
)
// To get clients to start as bootnodes, we pass in these values when starting them
var elClientContextForBootElClients *el.ELClientContext = nil
var clClientContextForBootClClients *cl.CLClientContext = nil

type ParticipantSpec struct {
	ELClientType ParticipantELClientType
	CLClientType ParticipantCLClientType
}

func LaunchParticipantNetwork(
	enclaveCtx *enclaves.EnclaveContext,
	networkId string,
	elClientLaunchers map[ParticipantELClientType]el.ELClientLauncher,
	clClientLaunchers map[ParticipantCLClientType]cl.CLClientLauncher,
	allParticipantSpecs []*ParticipantSpec,
	preregisteredValidatorKeysForNodes []*prelaunch_data_generator.NodeTypeKeystoreDirpaths,
	logLevel log_levels.ParticipantLogLevel,
) (
	[]*Participant,
	error,
) {
	participants := []*Participant{}
	for idx, participantSpec := range allParticipantSpecs {
		elClientType := participantSpec.ELClientType
		clClientType := participantSpec.CLClientType

		elLauncher, found := elClientLaunchers[elClientType]
		if !found {
			return nil, stacktrace.NewError("No EL client launcher defined for EL client type '%v'", elClientType)
		}
		clLauncher, found := clClientLaunchers[clClientType]
		if !found {
			return nil, stacktrace.NewError("No CL client launcher defined for CL client type '%v'", clClientType)
		}

		elClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", elClientServiceIdPrefix, idx))
		clClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", clClientServiceIdPrefix, idx))
		newClNodeValidatorKeystores := preregisteredValidatorKeysForNodes[idx]

		// Add EL client
		var newElClientCtx *el.ELClientContext
		var elClientLaunchErr error
		if idx == bootParticipantIndex {
			newElClientCtx, elClientLaunchErr = elLauncher.Launch(
				enclaveCtx,
				elClientServiceId,
				logLevel,
				networkId,
				elClientContextForBootElClients,
			)
		} else {
			bootParticipant := participants[bootParticipantIndex]
			bootElClientCtx := bootParticipant.GetELClientContext()
			newElClientCtx, elClientLaunchErr = elLauncher.Launch(
				enclaveCtx,
				elClientServiceId,
				logLevel,
				networkId,
				bootElClientCtx,
			)
		}
		if elClientLaunchErr != nil {
			return nil, stacktrace.Propagate(elClientLaunchErr, "An error occurred launching EL client for participant %v", idx)
		}

		// Launch CL client
		var newClClientCtx *cl.CLClientContext
		var clClientLaunchErr error
		if idx == bootParticipantIndex {
			newClClientCtx, clClientLaunchErr = clLauncher.Launch(
				enclaveCtx,
				clClientServiceId,
				logLevel,
				clClientContextForBootClClients,
				newElClientCtx,
				newClNodeValidatorKeystores,
			)
		} else {
			bootParticipant := participants[bootParticipantIndex]
			bootClClientCtx := bootParticipant.GetCLClientContext()
			newClClientCtx, clClientLaunchErr = clLauncher.Launch(
				enclaveCtx,
				clClientServiceId,
				logLevel,
				bootClClientCtx,
				newElClientCtx,
				newClNodeValidatorKeystores,
			)
		}
		if clClientLaunchErr != nil {
			return nil, stacktrace.Propagate(clClientLaunchErr, "An error occurred launching CL client for participant %v", idx)
		}

		participant := NewParticipant(
			elClientType,
			clClientType,
			newElClientCtx,
			newClClientCtx,
		)
		participants = append(participants, participant)
	}
	return participants, nil
}


/*
// Represents a network of virtual "participants", where each participant runs:
//  1) an EL client
//  2) a Beacon client
//  3) a validator client
type ParticipantNetwork struct {
	enclaveCtx *enclaves.EnclaveContext

	networkId string

	preregisteredValidatorKeysForNodes []*prelaunch_data_generator.NodeTypeKeystoreDirpaths

	participants []*Participant

	elClientLaunchers map[ParticipantELClientType]el.ELClientLauncher
	clClientLaunchers map[ParticipantCLClientType]cl.CLClientLauncher

	mutex *sync.Mutex
}

func NewParticipantNetwork(
	enclaveCtx *enclaves.EnclaveContext,
	networkId string,
	preregisteredValidatorKeysForNodes []*prelaunch_data_generator.NodeTypeKeystoreDirpaths,
	elClientLaunchers map[ParticipantELClientType]el.ELClientLauncher,
	clClientLaunchers map[ParticipantCLClientType]cl.CLClientLauncher,
) *ParticipantNetwork {
	return &ParticipantNetwork{
		enclaveCtx: enclaveCtx,
		networkId: networkId,
		preregisteredValidatorKeysForNodes: preregisteredValidatorKeysForNodes,
		participants: []*Participant{},
		elClientLaunchers: elClientLaunchers,
		clClientLaunchers: clClientLaunchers,
		mutex: &sync.Mutex{},
	}
}

func (network *ParticipantNetwork) AddParticipant(
	elClientType ParticipantELClientType,
	clClientType ParticipantCLClientType,
	logLevel log_levels.ParticipantLogLevel,
) (*Participant, error) {
	network.mutex.Lock()
	defer network.mutex.Unlock()

	elLauncher, found := network.elClientLaunchers[elClientType]
	if !found {
		return nil, stacktrace.NewError("No EL client launcher defined for EL client type '%v'", elClientType)
	}
	clLauncher, found := network.clClientLaunchers[clClientType]
	if !found {
		return nil, stacktrace.NewError("No CL client launcher defined for CL client type '%v'", clClientType)
	}

	newParticipantIdx := len(network.participants)
	elClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", elClientServiceIdPrefix, newParticipantIdx))
	clClientServiceId := services.ServiceID(fmt.Sprintf("%v%v", clClientServiceIdPrefix, newParticipantIdx))
	newClNodeValidatorKeystores := network.preregisteredValidatorKeysForNodes[newParticipantIdx]

	// Add EL client
	var newElClientCtx *el.ELClientContext
	var elClientLaunchErr error
	if newParticipantIdx == bootParticipantIndex {
		newElClientCtx, elClientLaunchErr = elLauncher.Launch(
			network.enclaveCtx,
			elClientServiceId,
			logLevel,
			network.networkId,
			elClientContextForBootElClients,
		)
	} else {
		bootParticipant := network.participants[bootParticipantIndex]
		bootElClientCtx := bootParticipant.GetELClientContext()
		newElClientCtx, elClientLaunchErr = elLauncher.Launch(
			network.enclaveCtx,
			elClientServiceId,
			logLevel,
			network.networkId,
			bootElClientCtx,
		)
	}
	if elClientLaunchErr != nil {
		return nil, stacktrace.Propagate(elClientLaunchErr, "An error occurred launching EL client for participant %v", newParticipantIdx)
	}

	// Launch CL client
	var newClClientCtx *cl.CLClientContext
	var clClientLaunchErr error
	if newParticipantIdx == bootParticipantIndex {
		newClClientCtx, clClientLaunchErr = clLauncher.Launch(
			network.enclaveCtx,
			clClientServiceId,
			logLevel,
			clClientContextForBootClClients,
			newElClientCtx,
			newClNodeValidatorKeystores,
		)
	} else {
		bootParticipant := network.participants[bootParticipantIndex]
		bootClClientCtx := bootParticipant.GetCLClientContext()
		newClClientCtx, clClientLaunchErr = clLauncher.Launch(
			network.enclaveCtx,
			clClientServiceId,
			logLevel,
			bootClClientCtx,
			newElClientCtx,
			newClNodeValidatorKeystores,
		)
	}
	if clClientLaunchErr != nil {
		return nil, stacktrace.Propagate(clClientLaunchErr, "An error occurred launching CL client for participant %v", newParticipantIdx)
	}

	participant := NewParticipant(
		elClientType,
		clClientType,
		newElClientCtx,
		newClClientCtx,
	)
	network.participants = append(network.participants, participant)

	return participant, nil
}

 */