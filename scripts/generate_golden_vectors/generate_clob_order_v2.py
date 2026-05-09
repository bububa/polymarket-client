#!/usr/bin/env python3
"""
Generate CLOB v2 golden vectors with Polymarket's official py-clob-client-v2.

This script intentionally does not call Polymarket network APIs.
It only uses py-clob-client-v2's local OrderBuilder to generate signed order
payloads for Go SDK golden tests.

Output:
  testdata/golden/py-clob-client-v2/clob_order_v2.json
"""

from __future__ import annotations

import argparse
import dataclasses
import json
import pathlib
from typing import Any, Dict

from py_clob_client_v2.clob_types import (
    CreateOrderOptions,
    MarketOrderArgsV2,
    OrderArgsV2,
    OrderType,
)
from py_clob_client_v2.order_builder.builder import OrderBuilder
import py_clob_client_v2.order_builder.builder as builder_mod
from py_clob_client_v2.order_utils import Side
from py_clob_client_v2.order_utils.model.order_data_v2 import order_to_json_v2
from py_clob_client_v2.order_utils.model.signature_type_v2 import SignatureTypeV2
from py_clob_client_v2.signer import Signer


# Deterministic public test key. Do not use on mainnet.
DEFAULT_PRIVATE_KEY = (
    "0x59c6995e998f97a5a0044966f094538092e1db9e7b9c0e5a4e9e4e9e4e9e4e9e"
)

# A large decimal token id shaped like a CLOB token id.
DEFAULT_TOKEN_ID = (
    "4738542302108129612856912335517660352849664845664685440963190764720214313804"
)

# Used as maker/funder for PROXY / SAFE signature modes.
DEFAULT_FUNDER = "0x1111111111111111111111111111111111111111"

DEFAULT_CHAIN_ID = 137
DEFAULT_TIMESTAMP = 1_700_000_000
DEFAULT_OWNER = "test-api-key-owner"
DEFAULT_TICK_SIZE = "0.01"


def as_jsonable(value: Any) -> Any:
    if dataclasses.is_dataclass(value):
        return {k: as_jsonable(v) for k, v in dataclasses.asdict(value).items()}
    if isinstance(value, dict):
        return {str(k): as_jsonable(v) for k, v in value.items()}
    if isinstance(value, (list, tuple)):
        return [as_jsonable(v) for v in value]
    if hasattr(value, "value"):
        return value.value
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


def build_limit_vector(
    *,
    private_key: str,
    chain_id: int,
    token_id: str,
    signature_type: SignatureTypeV2,
    funder: str | None,
    timestamp: int,
    owner: str,
    tick_size: str,
    neg_risk: bool,
    side: str,
    price: float,
    size: float,
    expiration: int,
    name: str,
) -> Dict[str, Any]:
    original_time = builder_mod.time.time
    builder_mod.time.time = lambda: timestamp
    try:
        signer = Signer(private_key, chain_id)
        builder = OrderBuilder(
            signer=signer,
            signature_type=signature_type,
            funder=funder,
        )

        args = OrderArgsV2(
            token_id=token_id,
            price=price,
            size=size,
            side=side,
            expiration=expiration,
        )
        options = CreateOrderOptions(tick_size=tick_size, neg_risk=neg_risk)

        signed = builder.build_order(args, options, version=2)
        post_body = order_to_json_v2(
            signed,
            owner=owner,
            order_type=OrderType.GTC,
            defer_exec=False,
        )

        return {
            "name": name,
            "kind": "clob_order_v2_limit",
            "generatedBy": {
                "sdk": "py-clob-client-v2",
                "script": "scripts/generate_golden_vectors/generate_clob_order_v2.py",
            },
            "input": {
                "privateKey": private_key,
                "chainId": chain_id,
                "tokenId": token_id,
                "signatureType": int(signature_type),
                "funder": funder,
                "timestamp": timestamp,
                "owner": owner,
                "tickSize": tick_size,
                "negRisk": neg_risk,
                "side": side,
                "price": price,
                "size": size,
                "expiration": expiration,
            },
            "expected": {
                "signer": signer.address(),
                "signedOrder": signed_order_to_dict(signed),
                "postOrderRequest": as_jsonable(post_body),
            },
        }
    finally:
        builder_mod.time.time = original_time


def build_market_vector(
    *,
    private_key: str,
    chain_id: int,
    token_id: str,
    signature_type: SignatureTypeV2,
    funder: str | None,
    timestamp: int,
    owner: str,
    tick_size: str,
    neg_risk: bool,
    side: str,
    amount: float,
    price: float,
    order_type: str,
    name: str,
) -> Dict[str, Any]:
    original_time = builder_mod.time.time
    builder_mod.time.time = lambda: timestamp
    try:
        signer = Signer(private_key, chain_id)
        builder = OrderBuilder(
            signer=signer,
            signature_type=signature_type,
            funder=funder,
        )

        args = MarketOrderArgsV2(
            token_id=token_id,
            amount=amount,
            side=side,
            price=price,
            order_type=order_type,
        )
        options = CreateOrderOptions(tick_size=tick_size, neg_risk=neg_risk)

        signed = builder.build_market_order(args, options, version=2)
        post_body = order_to_json_v2(
            signed,
            owner=owner,
            order_type=order_type,
            defer_exec=False,
        )

        return {
            "name": name,
            "kind": "clob_order_v2_market",
            "generatedBy": {
                "sdk": "py-clob-client-v2",
                "script": "scripts/generate_golden_vectors/generate_clob_order_v2.py",
            },
            "input": {
                "privateKey": private_key,
                "chainId": chain_id,
                "tokenId": token_id,
                "signatureType": int(signature_type),
                "funder": funder,
                "timestamp": timestamp,
                "owner": owner,
                "tickSize": tick_size,
                "negRisk": neg_risk,
                "side": side,
                "amount": amount,
                "price": price,
                "orderType": order_type,
            },
            "expected": {
                "signer": signer.address(),
                "signedOrder": signed_order_to_dict(signed),
                "postOrderRequest": as_jsonable(post_body),
            },
        }
    finally:
        builder_mod.time.time = original_time


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--out",
        default="testdata/golden/py-clob-client-v2/clob_order_v2.json",
    )
    parser.add_argument("--private-key", default=DEFAULT_PRIVATE_KEY)
    parser.add_argument("--chain-id", type=int, default=DEFAULT_CHAIN_ID)
    parser.add_argument("--token-id", default=DEFAULT_TOKEN_ID)
    parser.add_argument("--funder", default=DEFAULT_FUNDER)
    parser.add_argument("--owner", default=DEFAULT_OWNER)
    parser.add_argument("--timestamp", type=int, default=DEFAULT_TIMESTAMP)
    args = parser.parse_args()

    vectors = [
        build_limit_vector(
            private_key=args.private_key,
            chain_id=args.chain_id,
            token_id=args.token_id,
            signature_type=SignatureTypeV2.EOA,
            funder=None,
            timestamp=args.timestamp,
            owner=args.owner,
            tick_size=DEFAULT_TICK_SIZE,
            neg_risk=False,
            side="BUY",
            price=0.42,
            size=10.0,
            expiration=0,
            name="limit_buy_gtc_eoa",
        ),
        build_limit_vector(
            private_key=args.private_key,
            chain_id=args.chain_id,
            token_id=args.token_id,
            signature_type=SignatureTypeV2.POLY_PROXY,
            funder=args.funder,
            timestamp=args.timestamp,
            owner=args.owner,
            tick_size=DEFAULT_TICK_SIZE,
            neg_risk=False,
            side="BUY",
            price=0.42,
            size=10.0,
            expiration=0,
            name="limit_buy_gtc_proxy",
        ),
        build_limit_vector(
            private_key=args.private_key,
            chain_id=args.chain_id,
            token_id=args.token_id,
            signature_type=SignatureTypeV2.POLY_GNOSIS_SAFE,
            funder=args.funder,
            timestamp=args.timestamp,
            owner=args.owner,
            tick_size=DEFAULT_TICK_SIZE,
            neg_risk=False,
            side="SELL",
            price=0.58,
            size=12.0,
            expiration=0,
            name="limit_sell_gtc_safe",
        ),
        build_market_vector(
            private_key=args.private_key,
            chain_id=args.chain_id,
            token_id=args.token_id,
            signature_type=SignatureTypeV2.POLY_PROXY,
            funder=args.funder,
            timestamp=args.timestamp,
            owner=args.owner,
            tick_size=DEFAULT_TICK_SIZE,
            neg_risk=False,
            side="BUY",
            amount=25.0,
            price=0.42,
            order_type=OrderType.FOK,
            name="market_buy_fok_proxy",
        ),
        build_market_vector(
            private_key=args.private_key,
            chain_id=args.chain_id,
            token_id=args.token_id,
            signature_type=SignatureTypeV2.POLY_GNOSIS_SAFE,
            funder=args.funder,
            timestamp=args.timestamp,
            owner=args.owner,
            tick_size=DEFAULT_TICK_SIZE,
            neg_risk=False,
            side="SELL",
            amount=10.0,
            price=0.58,
            order_type=OrderType.FAK,
            name="market_sell_fak_safe",
        ),
    ]

    output = {
        "schema": "polymarket-client-golden-vectors/v1",
        "generatedBy": {
            "sdk": "py-clob-client-v2",
            "script": "scripts/generate_golden_vectors/generate_clob_order_v2.py",
        },
        "vectors": vectors,
    }

    out_path = pathlib.Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(output, indent=2, sort_keys=True) + "\n")
    print(f"wrote {out_path}")


if __name__ == "__main__":
    main()
