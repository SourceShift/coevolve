"""Normalized event envelope + provenance for the live run-flow.

One event vocabulary that every surface (the terminal live-flow, the web
Console, an SDK) renders identically — adopted from omnigent's
``external_conversation_item`` / ``external_session_status`` /
``external_session_usage`` normalization.

`Provenance` enforces the hard "all real data" rule at the type level: a value
carries where it came from, and the renderers refuse to show a bare number for
anything that is not measured. Nothing aspirational is ever displayed as real.
"""
from __future__ import annotations

import time
from dataclasses import dataclass, field
from enum import Enum
from typing import Any


class DataOrigin(str, Enum):
    REAL = "real"            # measured (DB, opencode usage, CN hit, verify verdict)
    ESTIMATED = "estimated"  # computed from real inputs (tokens × list price)
    NOT_YET = "not_yet"      # needs a capability we don't have (trained model)
    DEGRADED = "degraded"    # seam reachable but empty/offline right now


@dataclass(frozen=True)
class Provenance:
    origin: DataOrigin
    source: str = ""         # e.g. "llm_calls.cost_usd", "cn_capsule", "list-price"

    @property
    def is_real(self) -> bool:
        return self.origin is DataOrigin.REAL


def real(source: str) -> Provenance:
    return Provenance(DataOrigin.REAL, source)


def estimated(source: str) -> Provenance:
    return Provenance(DataOrigin.ESTIMATED, source)


def not_yet(source: str) -> Provenance:
    return Provenance(DataOrigin.NOT_YET, source)


@dataclass(frozen=True)
class Value:
    """A number/string carrying its provenance. Renderers gate on `.prov`."""
    value: Any
    prov: Provenance


class Route(str, Enum):
    LOCAL = "local"
    FRONTIER = "frontier"
    UNKNOWN = "unknown"


class EventType(str, Enum):
    RUN_STARTED = "run_started"
    NODE = "node"                # a mini-ork node step (classify/plan/execute/…)
    STATUS = "status"            # status change (running/escalated/done/failed)
    USAGE = "usage"              # cumulative cost/token tick
    RECALL = "recall"            # a ContextNest recall fed into the run
    REROUTE = "reroute"          # router auto-rerouted (quota/parity)
    PERMISSION = "permission"    # worker asked to perform a gated action
    LOG = "log"                  # a raw log line
    RUN_FINISHED = "run_finished"


@dataclass
class Event:
    """The single envelope. `data` holds type-specific fields (Values where
    they are numbers the UI shows, so provenance rides along)."""
    type: EventType
    run_id: str
    ts: float = field(default_factory=time.time)
    seq: int = 0                 # monotonic per run (SSE cursor / dedupe key)
    node: str = ""               # classify | plan | execute | verify | reflect | …
    text: str = ""
    data: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        def enc(v: Any) -> Any:
            if isinstance(v, Value):
                return {"value": v.value, "origin": v.prov.origin.value, "source": v.prov.source}
            if isinstance(v, Provenance):
                return {"origin": v.origin.value, "source": v.source}
            if isinstance(v, Enum):
                return v.value
            return v
        return {
            "type": self.type.value, "run_id": self.run_id, "ts": self.ts,
            "seq": self.seq, "node": self.node, "text": self.text,
            "data": {k: enc(v) for k, v in self.data.items()},
        }
