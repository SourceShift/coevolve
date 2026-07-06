"""Coevolve — the extensible CLI that runs the mini-ork loop with opencode as
the worker, shows the live flow + costs, and integrates ContextNest (memory)
and TraceOtter (learning).

Architecture (informed by omnigent, Apache-2.0):
  - `capabilities` — the feature matrix each integration declares.
  - `events` — one normalized event envelope + provenance ("all real data").
  - `integrations.base` — role ABCs (Orchestrator/Memory/Learning/Worker/Router).
  - `registry` — decorator + entry-point plugin discovery.
Everything else (seams, pipeline, run control, TUI, modes) programs against
these contracts, so backends and UI modes are additive.
"""
from __future__ import annotations

from .capabilities import Capabilities, IntegrationMode, Kind, Resume, Streaming
from .events import DataOrigin, Event, EventType, Provenance, Value, real, estimated, not_yet
from .integrations.base import (
    Health, Integration, Learning, Memory, Orchestrator, Recall, Router,
    RunContext, Worker, LaneChoice,
)
from . import registry

__all__ = [
    "Capabilities", "IntegrationMode", "Kind", "Resume", "Streaming",
    "DataOrigin", "Event", "EventType", "Provenance", "Value",
    "real", "estimated", "not_yet",
    "Health", "Integration", "Learning", "Memory", "Orchestrator", "Recall",
    "Router", "RunContext", "Worker", "LaneChoice", "registry",
]
