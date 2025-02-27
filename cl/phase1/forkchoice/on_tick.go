package forkchoice

import libcommon "github.com/tenderly/erigon/erigon-lib/common"

// OnTick executes on_tick operation for forkchoice.
func (f *ForkChoiceStore) OnTick(time uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tickSlot := (time - f.genesisTime) / f.beaconCfg.SecondsPerSlot
	for f.Slot() < tickSlot {
		previousTime := f.genesisTime + (f.Slot()+1)*f.beaconCfg.SecondsPerSlot
		f.onTickPerSlot(previousTime)
	}
	f.onTickPerSlot(time)
}

// onTickPerSlot handles ticks
func (f *ForkChoiceStore) onTickPerSlot(time uint64) {
	previousSlot := f.Slot()
	f.time = time
	currentSlot := f.Slot()
	if currentSlot <= previousSlot {
		return
	}
	f.headHash = libcommon.Hash{}
	// If this is a new slot, reset store.proposer_boost_root
	f.proposerBoostRoot = libcommon.Hash{}
	if f.computeSlotsSinceEpochStart(currentSlot) == 0 {
		f.updateCheckpoints(f.unrealizedJustifiedCheckpoint.Copy(), f.unrealizedFinalizedCheckpoint.Copy())
	}
}
