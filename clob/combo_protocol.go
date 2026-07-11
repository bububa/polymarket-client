package clob

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
)

var uint256Max = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

// CanonicalizeComboLegs validates, sorts, and defensively copies Combo leg IDs.
func CanonicalizeComboLegs(legs []string) ([]*big.Int, error) {
	if len(legs) == 0 || len(legs) > MaxComboLegs {
		return nil, fmt.Errorf("polymarket: combo legs must include 1 to %d position ids", MaxComboLegs)
	}
	positions := make([]*big.Int, len(legs))
	for i, leg := range legs {
		position, err := parseComboPositionID(leg)
		if err != nil {
			return nil, fmt.Errorf("polymarket: combo leg %d: %w", i, err)
		}
		encoded := position.FillBytes(make([]byte, 32))
		if (encoded[0] != 1 && encoded[0] != 2) || encoded[31] > 1 {
			return nil, errors.New("polymarket: combo legs must be binary or neg-risk yes/no position ids")
		}
		positions[i] = position
	}
	sort.Slice(positions, func(i, j int) bool { return positions[i].Cmp(positions[j]) < 0 })
	for i := 1; i < len(positions); i++ {
		previous := positions[i-1].FillBytes(make([]byte, 32))
		current := positions[i].FillBytes(make([]byte, 32))
		if positions[i-1].Cmp(positions[i]) == 0 {
			return nil, errors.New("polymarket: combo legs must not contain duplicate position ids")
		}
		if string(previous[:31]) == string(current[:31]) {
			return nil, errors.New("polymarket: combo legs must not contain both outcomes for the same condition")
		}
	}
	return positions, nil
}

// DeriveComboPositionContext derives the Combo condition and complementary IDs.
func DeriveComboPositionContext(legs []string) (ComboPositionContext, error) {
	canonical, err := CanonicalizeComboLegs(legs)
	if err != nil {
		return ComboPositionContext{}, err
	}
	uintArray, err := abi.NewType("uint256[]", "", nil)
	if err != nil {
		return ComboPositionContext{}, fmt.Errorf("polymarket: create combo legs abi type: %w", err)
	}
	encodedLegs, err := (abi.Arguments{{Type: uintArray}}).Pack(canonical)
	if err != nil {
		return ComboPositionContext{}, fmt.Errorf("polymarket: encode combo legs: %w", err)
	}
	uintType, _ := abi.NewType("uint256", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	base, err := (abi.Arguments{{Type: uintType}, {Type: bytesType}}).Pack(big.NewInt(3), encodedLegs)
	if err != nil {
		return ComboPositionContext{}, fmt.Errorf("polymarket: encode combo condition: %w", err)
	}
	hash := crypto.Keccak256(base)
	var condition ComboConditionID
	condition[0] = 3
	copy(condition[1:17], hash[16:])
	return ComboPositionContext{
		ConditionID: condition,
		PositionIDs: [2]string{
			new(big.Int).SetBytes(append(append([]byte(nil), condition[:]...), 0)).String(),
			new(big.Int).SetBytes(append(append([]byte(nil), condition[:]...), 1)).String(),
		},
	}, nil
}

// DecodeComboPositionID returns the embedded condition and outcome (0=YES, 1=NO).
func DecodeComboPositionID(positionID string) (ComboConditionID, uint8, error) {
	position, err := parseComboPositionID(positionID)
	if err != nil {
		return ComboConditionID{}, 0, err
	}
	encoded := position.FillBytes(make([]byte, 32))
	if encoded[0] != 3 {
		return ComboConditionID{}, 0, errors.New("polymarket: combo position id must use the combinatorial module")
	}
	if encoded[31] > 1 {
		return ComboConditionID{}, 0, errors.New("polymarket: combo position id must be a yes/no position id")
	}
	var condition ComboConditionID
	copy(condition[:], encoded[:31])
	return condition, encoded[31], nil
}

func parseComboPositionID(value string) (*big.Int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("position id must be a uint256 value")
	}
	position, ok := new(big.Int).SetString(value, 10)
	if !ok || position.Sign() < 0 || position.Cmp(uint256Max) > 0 {
		return nil, errors.New("position id must be a uint256 value")
	}
	return position, nil
}
