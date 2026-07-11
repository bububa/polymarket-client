package clob

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const comboABIJSON = `[
  {"type":"function","name":"prepareCondition","stateMutability":"nonpayable","inputs":[{"name":"legs","type":"uint256[]"}],"outputs":[{"name":"conditionId","type":"bytes31"}]},
  {"type":"function","name":"split","stateMutability":"nonpayable","inputs":[{"name":"conditionId","type":"bytes31"},{"name":"amount","type":"uint256"}],"outputs":[]},
  {"type":"function","name":"merge","stateMutability":"nonpayable","inputs":[{"name":"conditionId","type":"bytes31"},{"name":"amount","type":"uint256"}],"outputs":[]},
  {"type":"function","name":"redeem","stateMutability":"nonpayable","inputs":[{"name":"conditionId","type":"bytes31"},{"name":"outcomeIndex","type":"uint256"},{"name":"amount","type":"uint256"}],"outputs":[]}
]`

const comboPositionManagerABIJSON = `[
  {"type":"function","name":"balanceOf","stateMutability":"view","inputs":[{"name":"account","type":"address"},{"name":"id","type":"uint256"}],"outputs":[{"name":"balance","type":"uint256"}]},
  {"type":"function","name":"balanceOfBatch","stateMutability":"view","inputs":[{"name":"accounts","type":"address[]"},{"name":"ids","type":"uint256[]"}],"outputs":[{"name":"balances","type":"uint256[]"}]}
]`

const comboApprovalABIJSON = `[
  {"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"success","type":"bool"}]},
  {"type":"function","name":"setApprovalForAll","stateMutability":"nonpayable","inputs":[{"name":"operator","type":"address"},{"name":"approved","type":"bool"}],"outputs":[]}
]`

var comboABI = mustParseComboABI(comboABIJSON)
var comboPositionManagerABI = mustParseComboABI(comboPositionManagerABIJSON)
var comboApprovalABI = mustParseComboABI(comboApprovalABIJSON)

func mustParseComboABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}
