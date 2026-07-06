"""Integration contracts — the pluggable seams of the Coevolve loop.

Every backend (mini-ork orchestration, ContextNest memory, TraceOtter learning,
opencode worker, the lane router) implements one of the role ABCs below and
self-registers. Core code programs against these ABCs and the declared
``Capabilities`` only — never against a concrete backend — so a new backend is
additive. Modeled on omnigent's Executor + HarnessContribution split.
"""
from __future__ import annotations

import abc
from collections.abc import AsyncIterator, Iterable
from dataclasses import dataclass, field
from typing import Any

from ..capabilities import Capabilities
from ..events import Event, Provenance


@dataclass
class Health:
    available: bool
    detail: str = ""            # e.g. "ContextNest offline", "cold-start"
    degraded: bool = False


@dataclass
class Recall:
    text: str
    source: str                 # basin / session / capsule id
    atom_id: str = ""
    prov: Provenance | None = None


@dataclass
class LaneChoice:
    lane: str                   # config/agents.yaml lane name
    model: str                  # resolved provider/model for the worker
    advantage: float | None     # relative_advantage from the router (None=cold)
    runs_count: int = 0
    prov: Provenance | None = None


@dataclass
class RunContext:
    """Everything a run needs, assembled by core before dispatch."""
    task: str
    task_class: str = ""
    objective_domain: str = "eng-team"
    run_dir: str = ""
    recalls: list[Recall] = field(default_factory=list)
    lane: LaneChoice | None = None
    dry_run: bool = False
    extras: dict[str, Any] = field(default_factory=dict)


class Integration(abc.ABC):
    """Base for all pluggable backends."""
    name: str = ""              # unique key, e.g. "mini-ork", "opencode"

    @abc.abstractmethod
    def capabilities(self) -> Capabilities: ...

    @abc.abstractmethod
    def health(self) -> Health: ...


class Orchestrator(Integration):
    """Drives a run through the loop, emitting normalized events. (mini-ork)"""
    @abc.abstractmethod
    async def run(self, ctx: RunContext) -> AsyncIterator[Event]: ...

    @abc.abstractmethod
    def stop(self, run_id: str) -> bool: ...

    @abc.abstractmethod
    def resume(self, run_id: str) -> bool: ...

    @abc.abstractmethod
    def rollback(self, run_id: str) -> bool: ...

    @abc.abstractmethod
    def list_runs(self, limit: int = 20) -> list[dict[str, Any]]: ...


class Memory(Integration):
    """Recall + outcome write-back. (ContextNest)"""
    @abc.abstractmethod
    def recall(self, query: str, limit: int = 8) -> list[Recall]: ...

    @abc.abstractmethod
    def write_outcome(self, outcome: str, atom_ids: Iterable[str],
                      evidence: str = "") -> bool: ...


class Learning(Integration):
    """Distill traces into a training set / skills. (TraceOtter)"""
    @abc.abstractmethod
    def distill(self, runs_dir: str) -> dict[str, Any]: ...


class Worker(Integration):
    """Executes a node's actual work against a model. (opencode)

    The batch path already exists as lib/providers/cl_opencode.sh; a live path
    (opencode serve + SSE) plugs in here later. Core selects a Worker by
    Capabilities.models ∋ chosen model."""
    def supports_model(self, model: str) -> bool:
        return model in self.capabilities().models


class Router(Integration):
    """Chooses lane+model per node from learned advantage. (lane_router)"""
    @abc.abstractmethod
    def pick_lane(self, task_class: str, node_type: str,
                  objective_domain: str, code_region: str = "") -> LaneChoice: ...

    @abc.abstractmethod
    def recompute(self) -> None: ...
