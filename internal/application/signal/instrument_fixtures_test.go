package signal

import "internal/domain/instrument"

// btcUSDTPerp / ethUSDTPerp are the canonical fixtures for the
// internal-package sampler tests after H-6.c.1 commit 7a removed the
// legacy (source, symbol) constructors. The Perpetual contract
// reflects the original ("binancef", "btcusdt") tuple semantics
// produced by the H-6.b sunset boundary helper.
var (
	btcUSDTPerp = func() instrument.CanonicalInstrument {
		inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
		if prob != nil {
			panic("test setup: BTC/USDT-perp: " + prob.Message)
		}
		return inst
	}()

	ethUSDTPerp = func() instrument.CanonicalInstrument {
		inst, prob := instrument.New("ETH", "USDT", instrument.ContractPerpetual)
		if prob != nil {
			panic("test setup: ETH/USDT-perp: " + prob.Message)
		}
		return inst
	}()
)
