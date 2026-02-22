# Change: Backend Clean Architecture Refactoring (Phases B-D)

## Why
Phase A (entities/dto separation) is complete. The backend still has handlers calling stores directly, ports mixed with adapters, and no services layer. This violates the Clean/Hexagonal Architecture target and makes it harder to add new features, test in isolation, and onboard new developers.

## What Changes
- **Phase B**: Extract `ports/` (interfaces) and `adapters/` (postgres + memory implementations) from `store/`
- **Phase C**: Introduce `services/` layer between handlers and ports (biggest change)
- **Phase D**: Extract validators, remove backward-compat wrappers, final dependency audit

## Impact
- Affected specs: none (pure refactoring, no behavior changes)
- Affected code: `services/api/internal/` — restructuring of store/, handlers/, new services/, ports/, adapters/ directories
- **No breaking API changes** — all endpoints behave identically
- Risk: Phase C (services layer) requires careful method-by-method extraction
