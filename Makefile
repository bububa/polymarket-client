PYTHON_GOLDEN ?= python3.11
UNAME_S := $(shell uname -s)

GOLDEN_RUN_ENV := env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy

ifeq ($(UNAME_S),Darwin)
SDKROOT := $(shell xcrun --sdk macosx --show-sdk-path 2>/dev/null)
GOLDEN_ENV := SDKROOT="$(SDKROOT)" CFLAGS="-isysroot $(SDKROOT) -I$(SDKROOT)/usr/include" CPPFLAGS="-isysroot $(SDKROOT) -I$(SDKROOT)/usr/include" LDFLAGS="-isysroot $(SDKROOT) -L$(SDKROOT)/usr/lib"
else
GOLDEN_ENV :=
endif

.PHONY: golden

golden:
	rm -rf .venv-golden
	$(PYTHON_GOLDEN) -m venv .venv-golden
	. .venv-golden/bin/activate && python -m pip install -U pip setuptools wheel
	. .venv-golden/bin/activate && $(GOLDEN_ENV) pip install -r scripts/generate_golden_vectors/requirements.txt
	. .venv-golden/bin/activate && $(GOLDEN_RUN_ENV) python scripts/generate_golden_vectors/generate_clob_order_v2.py
	. .venv-golden/bin/activate && $(GOLDEN_RUN_ENV) python scripts/generate_golden_vectors/generate_relayer_safe.py
	. .venv-golden/bin/activate && $(GOLDEN_RUN_ENV) python scripts/generate_golden_vectors/generate_relayer_proxy.py
	. .venv-golden/bin/activate && $(GOLDEN_RUN_ENV) python scripts/generate_golden_vectors/generate_deposit_wallet_order_v2.py
