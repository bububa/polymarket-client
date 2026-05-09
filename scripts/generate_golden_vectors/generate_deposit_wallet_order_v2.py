#!/usr/bin/env python3
"""
Generate POLY_1271 / deposit wallet CLOB v2 golden vector with
Polymarket's official py-clob-client-v2.

This script intentionally does not call Polymarket network APIs.

Output:
  testdata/golden/py-clob-client-v2/clob_order_v2_deposit_wallet.json
"""

from __future__ import annotations

import argparse
import dataclasses
import json
import pathlib
from typing import Any, Dict

from py_clob_client_v2.config import get_contract_config
from py_clob_client_v2.constants import BYTES32_ZERO
from py_clob_client_v2.order_utils import ExchangeOrderBuilderV2, Side
from py_clob_client_v2.order_utils.model.order_data_v2 import (
    OrderDataV2,
    order_to_json_v2,
)
from py_clob_client_v2.order_utils.model.signature_type_v2 import SignatureTypeV2
from py_clob_client_v2.clob_types import OrderType
from py_clob_client_v2.signer import Signer


# Deterministic public test key. Do not use on mainnet.
DEFAULT_PRIVATE_KEY = (
    "0x59c6995e998f97a5a0044966f094538092e1db9e7b9c0e5a4e9e4e9e4e9e4e9e"
)

DEFAULT_TOKEN_ID = (
    "4738542302108129612856912335517660352849664845664685440963190764720214313804"
)

# Deposit wallet / POLY_1271 maker+signer.
DEFAULT_DEPOSIT_WALLET = "0x1111111111111111111111111111111111111111"

DEFAULT_OWNER = "test-api-key-owner"
DEFAULT_CHAIN_ID = 137
DEFAULT_TIMESTAMP = "1700000000000"
DEFAULT_SALT = 12345


def as_jsonable(value: Any) -> Any:
    if dataclasses.is_dataclass(value):
        return {k: as_jsonable(v) for k, v in dataclasses.asdict(value).items()}
    if isinstance(value, dict):
        return {str(k): as_jsonable(v) for k, v in value.items()}
    if isinstance(value, (list, tuple)):
        return [as_jsonable(v) for v in value]
    if isinstance(value, bytes):
        return "0x" + value.hex()
    if hasattr(value, "value"):
        return as_jsonable(value.value)
    return value


def signed_order_to_dict(order: Any) -> Dict[str, Any]:
    return {
        "salt": str(order.salt),
        "maker": order.maker,
        "signer": order.signer,
        "tokenId": order.tokenId,
        "makerAmount": order.makerAmount,
        "takerAmount": order.takerAmount,
        "side": int(order.side),
        "signatureType": int(order.signatureType),
        "timestamp": order.timestamp,
        "metadata": order.metadata,
        "builder": order.builder,
        "expiration": order.expiration,
        "signature": order.signature,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--out",
        default="testdata/golden/py-clob-client-v2/clob_order_v2_deposit_wallet.json",
    )
    parser.add_argument("--private-key", default=DEFAULT_PRIVATE_KEY)
    parser.add_argument("--chain-id", type=int, default=DEFAULT_CHAIN_ID)
    parser.add_argument("--token-id", default=DEFAULT_TOKEN_ID)
    parser.add_argument("--deposit-wallet", default=DEFAULT_DEPOSIT_WALLET)
    parser.add_argument("--owner", default=DEFAULT_OWNER)
    parser.add_argument("--timestamp", default=DEFAULT_TIMESTAMP)
    parser.add_argument("--salt", type=int, default=DEFAULT_SALT)
    args = parser.parse_args()

    signer = Signer(args.private_key, args.chain_id)
    contract_config = get_contract_config(args.chain_id)

    # py-clob-client-v2 v2 exchange address.
    exchange_address = contract_config.exchange_v2

    builder = ExchangeOrderBuilderV2(
        exchange_address,
        args.chain_id,
        signer,
        generate_salt=lambda: args.salt,
    )

    order_data = OrderDataV2(
        maker=args.deposit_wallet,
        signer=args.deposit_wallet,
        tokenId=args.token_id,
        makerAmount="4200000",
        takerAmount="10000000",
        side=Side.BUY,
        signatureType=SignatureTypeV2.POLY_1271,
        timestamp=args.timestamp,
        metadata=BYTES32_ZERO,
        builder=BYTES32_ZERO,
        expiration="0",
    )

    signed = builder.build_signed_order(order_data)
    typed_data = builder.build_order_typed_data(signed)
    order_hash = builder.build_order_hash(typed_data)

    post_body = order_to_json_v2(
        signed,
        owner=args.owner,
        order_type=OrderType.GTC,
        defer_exec=False,
    )

    output = {
        "schema": "polymarket-client-golden-vectors/v1",
        "name": "clob_order_v2_deposit_wallet_poly1271_limit_buy_gtc",
        "kind": "clob_order_v2_deposit_wallet",
        "generatedBy": {
            "sdk": "py-clob-client-v2",
            "script": "scripts/generate_golden_vectors/generate_deposit_wallet_order_v2.py",
        },
        "input": {
            "privateKey": args.private_key,
            "chainId": args.chain_id,
            "tokenId": args.token_id,
            "depositWallet": args.deposit_wallet,
            "signatureType": int(SignatureTypeV2.POLY_1271),
            "owner": args.owner,
            "timestamp": args.timestamp,
            "salt": str(args.salt),
            "side": "BUY",
            "makerAmount": "4200000",
            "takerAmount": "10000000",
            "expiration": "0",
            "metadata": BYTES32_ZERO,
            "builder": BYTES32_ZERO,
            "exchange": exchange_address,
        },
        "expected": {
            "ownerSigner": signer.address(),
            "depositWallet": args.deposit_wallet,
            "orderHash": order_hash,
            "typedData": as_jsonable(typed_data),
            "signedOrder": signed_order_to_dict(signed),
            "postOrderRequest": as_jsonable(post_body),
        },
    }

    out_path = pathlib.Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(output, indent=2, sort_keys=True) + "\n")
    print(f"wrote {out_path}")


if __name__ == "__main__":
    main()
