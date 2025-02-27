package handler

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/tenderly/erigon/cl/beacon/beaconhttp"
	"github.com/tenderly/erigon/cl/beacon/synced_data"
	"github.com/tenderly/erigon/cl/clparams"
	"github.com/tenderly/erigon/cl/persistence"
	"github.com/tenderly/erigon/cl/phase1/forkchoice"
	"github.com/tenderly/erigon/cl/pool"
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/turbo/snapshotsync/freezeblocks"
)

type ApiHandler struct {
	o   sync.Once
	mux chi.Router

	blockReader     freezeblocks.BeaconSnapshotReader
	indiciesDB      kv.RoDB
	genesisCfg      *clparams.GenesisConfig
	beaconChainCfg  *clparams.BeaconChainConfig
	forkchoiceStore forkchoice.ForkChoiceStorage
	operationsPool  pool.OperationsPool
	syncedData      *synced_data.SyncedDataManager
}

func NewApiHandler(genesisConfig *clparams.GenesisConfig, beaconChainConfig *clparams.BeaconChainConfig, source persistence.RawBeaconBlockChain, indiciesDB kv.RoDB, forkchoiceStore forkchoice.ForkChoiceStorage, operationsPool pool.OperationsPool, rcsn freezeblocks.BeaconSnapshotReader, syncedData *synced_data.SyncedDataManager) *ApiHandler {
	return &ApiHandler{o: sync.Once{}, genesisCfg: genesisConfig, beaconChainCfg: beaconChainConfig, indiciesDB: indiciesDB, forkchoiceStore: forkchoiceStore, operationsPool: operationsPool, blockReader: rcsn, syncedData: syncedData}
}

func (a *ApiHandler) init() {
	r := chi.NewRouter()
	a.mux = r
	// This is the set of apis for validation + otterscan
	// otterscn specific ones are commented as such
	r.Route("/eth", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Get("/events", http.NotFound)
			r.Route("/config", func(r chi.Router) {
				r.Get("/spec", beaconhttp.HandleEndpointFunc(a.getSpec))
				r.Get("/deposit_contract", beaconhttp.HandleEndpointFunc(a.getDepositContract))
				r.Get("/fork_schedule", beaconhttp.HandleEndpointFunc(a.getForkSchedule))
			})
			r.Route("/beacon", func(r chi.Router) {
				r.Route("/headers", func(r chi.Router) {
					r.Get("/", beaconhttp.HandleEndpointFunc(a.getHeaders))
					r.Get("/{block_id}", beaconhttp.HandleEndpointFunc(a.getHeader))
				})
				r.Route("/blocks", func(r chi.Router) {
					r.Post("/", http.NotFound)
					r.Get("/{block_id}", beaconhttp.HandleEndpointFunc(a.getBlock))
					r.Get("/{block_id}/attestations", beaconhttp.HandleEndpointFunc(a.getBlockAttestations))
					r.Get("/{block_id}/root", beaconhttp.HandleEndpointFunc(a.getBlockRoot))
				})
				r.Get("/genesis", beaconhttp.HandleEndpointFunc(a.getGenesis))
				r.Post("/binded_blocks", http.NotFound)
				r.Route("/pool", func(r chi.Router) {
					r.Post("/attestations", http.NotFound)
					r.Get("/voluntary_exits", beaconhttp.HandleEndpointFunc(a.poolVoluntaryExits))
					r.Get("/attester_slashings", beaconhttp.HandleEndpointFunc(a.poolAttesterSlashings))
					r.Get("/proposer_slashings", beaconhttp.HandleEndpointFunc(a.poolProposerSlashings))
					r.Get("/bls_to_execution_changes", beaconhttp.HandleEndpointFunc(a.poolBlsToExecutionChanges))
					r.Get("/attestations", beaconhttp.HandleEndpointFunc(a.poolAttestations))
					r.Post("/sync_committees", http.NotFound)
				})
				r.Get("/node/syncing", http.NotFound)
				r.Route("/states", func(r chi.Router) {
					r.Get("/head/validators/{index}", http.NotFound) // otterscan
					r.Get("/head/committees", http.NotFound)         // otterscan
					r.Route("/{state_id}", func(r chi.Router) {
						r.Get("/validators", http.NotFound)
						r.Get("/root", beaconhttp.HandleEndpointFunc(a.getStateRoot))
						r.Get("/fork", beaconhttp.HandleEndpointFunc(a.getStateFork))
						r.Get("/validators/{id}", http.NotFound)
					})
				})
			})
			r.Route("/validator", func(r chi.Router) {
				r.Route("/duties", func(r chi.Router) {
					r.Post("/attester/{epoch}", http.NotFound)
					r.Get("/proposer/{epoch}", beaconhttp.HandleEndpointFunc(a.getDutiesProposer))
					r.Post("/sync/{epoch}", http.NotFound)
				})
				r.Get("/blinded_blocks/{slot}", http.NotFound)
				r.Get("/attestation_data", http.NotFound)
				r.Get("/aggregate_attestation", http.NotFound)
				r.Post("/aggregate_and_proofs", http.NotFound)
				r.Post("/beacon_committee_subscriptions", http.NotFound)
				r.Post("/sync_committee_subscriptions", http.NotFound)
				r.Get("/sync_committee_contribution", http.NotFound)
				r.Post("/contribution_and_proofs", http.NotFound)
				r.Post("/prepare_beacon_proposer", http.NotFound)
			})
		})
		r.Route("/v2", func(r chi.Router) {
			r.Route("/debug", func(r chi.Router) {
				r.Route("/beacon", func(r chi.Router) {
					r.Get("/states/{state_id}", beaconhttp.HandleEndpointFunc(a.getFullState))
				})
			})
			r.Route("/beacon", func(r chi.Router) {
				r.Get("/blocks/{block_id}", beaconhttp.HandleEndpointFunc(a.getBlock)) //otterscan
			})
			r.Route("/validator", func(r chi.Router) {
				r.Post("/blocks/{slot}", http.NotFound)
			})
		})
	})
}

func (a *ApiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.o.Do(func() {
		a.init()
	})
	a.mux.ServeHTTP(w, r)
}
