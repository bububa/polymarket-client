#!/usr/bin/env python3
"""
Generate SAFE-mode CLOB v2 golden vectors using py-clob-client-v2.

Despite the filename, this is not a builder-relayer/gasless submit vector.
It verifies CLOB v2 signing/request data with SignatureTypeV2.POLY_GNOSIS_SAFE.
"""

from __future__ import annotations

import argparse
import json
import pathlib

from py_clob_client_v2.clob_types import CreateOrderOptions, OrderArgsV2, OrderType
from py_clob_client_v2.order_builder.builder import OrderBuilder
import py_clob_client_v2.order_builder.builder as builder_mod
from py_clob_client_v2.order_utils.model.order_data_v2 import order_to_json_v2
from py_clob_client_v2.order_utils.model.signature_type_v2 import SignatureTypeV2
from py_clob_client_v2.signer import Signer


DEFAULT_PRIVATE_KEY = (
    "0x59c6995e998f97a5a0044966f094538092e1db9e7b9c0e5a4e9e4e9e4e9e4e9e"
)
DEFAULT_TOKEN_ID = (
    "4738542302108129612856912335517660352849664845664685440963190764720214313804"
)
DEFAULT_FUNDER = "0x1111111111111111111111111111111111111111"
DEFAULT_OWNER = "test-api-key-owner"
DEFAULT_CHAIN_ID = 137
DEFAULT_TIMESTAMP = 1_700_000_000


def signed_order_to_dict(order):
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
        default="testdata/golden/py-clob-client-v2/clob_order_v2_safe.json",
    )
    parser.add_argument("--private-key", default=DEFAULT_PRIVATE_KEY)
    parser.add_argument("--chain-id", type=int, default=DEFAULT_CHAIN_ID)
    parser.add_argument("--token-id", default=DEFAULT_TOKEN_ID)
    parser.add_argument("--funder", default=DEFAULT_FUNDER)
    parser.add_argument("--owner", default=DEFAULT_OWNER)
    parser.add_argument("--timestamp", type=int, default=DEFAULT_TIMESTAMP)
    args = parser.parse_args()

    original_time = builder_mod.time.time
    builder_mod.time.time = lambda: args.timestamp
    try:
        signer = Signer(args.private_key, args.chain_id)
        builder = OrderBuilder(
            signer=signer,
            signature_type=SignatureTypeV2.POLY_GNOSIS_SAFE,
            funder=args.funder,
        )

        order_args = OrderArgsV2(
            token_id=args.token_id,
            price=0.58,
            size=12.0,
            side="SELL",
            expiration=0,
        )
        options = CreateOrderOptions(tick_size="0.01", neg_risk=False)

        signed = builder.build_order(order_args, options, version=2)
        post_body = order_to_json_v2(
            signed,
            owner=args.owner,
            order_type=OrderType.GTC,
            defer_exec=False,
        )

        output = {
            "schema": "polymarket-client-golden-vectors/v1",
            "name": "clob_order_v2_safe_limit_sell_gtc",
            "generatedBy": {
                "sdk": "py-clob-client-v2",
                "script": "scripts/generate_golden_vectors/generate_relayer_safe.py",
            },
            "input": {
                "privateKey": args.private_key,
                "chainId": args.chain_id,
                "tokenId": args.token_id,
                "signatureType": int(SignatureTypeV2.POLY_GNOSIS_SAFE),
                "funder": args.funder,
                "owner": args.owner,
                "timestamp": args.timestamp,
                "tickSize": "0.01",
                "negRisk": False,
                "side": "SELL",
                "price": 0.58,
                "size": 12.0,
                "expiration": 0,
            },
            "expected": {
                "signer": signer.address(),
                "signedOrder": signed_order_to_dict(signed),
                "postOrderRequest": post_body,
            },
        }
    finally:
        builder_mod.time.time = original_time

    out_path = pathlib.Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(output, indent=2, sort_keys=True) + "\n")
    print(f"wrote {out_path}")


if __name__ == "__main__":
    main()
