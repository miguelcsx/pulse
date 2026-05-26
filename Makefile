.PHONY: local demo dev dev-stone dev-ember dev-grove build test migrate seed seed-reset clean services services-stop services-status

# ── Infrastructure (Nix) ─────────────────────────────────────────────────────
services:
	nix run ./roots#roots-services -- start

services-stop:
	nix run ./roots#roots-services -- stop

services-status:
	nix run ./roots#roots-services -- status

# ── Backend ──────────────────────────────────────────────────────────────────
dev-stone:
	nix run ./stone#stone-dev

build-stone:
	nix run ./stone#stone-build

migrate:
	nix run ./stone#stone-migrate -- $(ARGS)

seed:
	nix run ./stone#stone-seed -- $(ARGS)

seed-reset:
	nix run ./stone#stone-seed -- --reset

# ── Frontend ─────────────────────────────────────────────────────────────────
dev-ember:
	nix run ./ember#ember-dev

dev-grove:
	nix run ./grove#grove-dev

# ── All-in-one ───────────────────────────────────────────────────────────────
local:
	nix run ./stone#stone-local

demo: services migrate seed-reset local

dev:
	@echo "Run in separate terminals:"
	@echo "  make services       (start PostgreSQL + Redis via Nix)"
	@echo "  make dev-stone      (backend on :8080)"
	@echo "  make dev-ember      (PWA on :5173)"
	@echo "  make dev-grove      (landing on :5174)"
	@echo ""
	@echo "Or run everything at once:"
	@echo "  make local          (mprocs TUI — all services)"

install:
	cd drift/ts && npm install
	cd ember && npm install
	cd grove && npm install

build: build-stone
	nix run ./ember#ember-build
	nix run ./grove#grove-build

# ── Tests ────────────────────────────────────────────────────────────────────
test: test-stone test-ember test-grove
	@echo "All tests passed"

test-stone:
	nix run ./stone#stone-test

test-ember:
	nix run ./ember#ember-lint
	nix run ./ember#ember-build

test-grove:
	nix run ./grove#grove-lint
	nix run ./grove#grove-build

# ── Nix dev shells ───────────────────────────────────────────────────────────
shell-roots:
	cd roots && nix develop

shell-stone:
	cd stone && nix develop

shell-ember:
	cd ember && nix develop

shell-grove:
	cd grove && nix develop

# ── Cleanup ──────────────────────────────────────────────────────────────────
clean:
	rm -rf stone/bin
	rm -rf ember/dist
	rm -rf grove/dist
	rm -rf .data
