"""Capability declarations for Coevolve integrations.

Adopted from omnigent's `HarnessCapabilities` pattern (Apache-2.0): every
integration declares WHAT it can do as data, so the core (and the UI) can adapt
without hard-coding per-integration branches. This is the backbone of the
"proper extensible way" — a new worker/memory/learning backend ships a
``Capabilities`` row and self-registers; nothing in core changes.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum


class IntegrationMode(str, Enum):
    """How the core talks to the integration."""
    IN_PROCESS = "in_process"        # a Python import (mini_ork.*, cn_client.py)
    CLI_SUBPROCESS = "cli_subprocess"  # shell out to bin/* or lib/*.sh (batch)
    NATIVE_SERVER = "native_server"  # a long-lived HTTP+SSE server (opencode serve)
    HTTP_API = "http_api"            # a remote REST service (ContextNest)


class Resume(str, Enum):
    NONE = "none"
    COLD_ONLY = "cold_only"          # resume by replaying transcript/plan
    WARM_REATTACH = "warm_reattach"  # reattach to a live session/cursor


class Streaming(str, Enum):
    NONE = "none"
    EVENTS = "events"                # emits normalized events as work proceeds
    SSE = "sse"                      # native server-sent-event stream to bridge


class Kind(str, Enum):
    """The role an integration plays in the loop."""
    ORCHESTRATOR = "orchestrator"    # mini-ork: drives classify→…→reflect
    MEMORY = "memory"                # ContextNest: recall + write-back
    LEARNING = "learning"            # TraceOtter: distill traces → training set
    WORKER = "worker"                # opencode: executes a node's actual work
    ROUTER = "router"                # lane/model selection


@dataclass(frozen=True)
class Capabilities:
    """The declared feature matrix for one integration."""
    kind: Kind
    mode: IntegrationMode
    streaming: Streaming = Streaming.NONE
    resume: Resume = Resume.NONE
    # discrete abilities the UI/core key off:
    interrupt: bool = False          # can a run be stopped mid-flight?
    cost_reporting: bool = False     # emits real per-call cost
    permissions: bool = False        # gates tool actions through policy
    subagents: bool = False          # can spawn child runs
    models: tuple[str, ...] = ()     # concrete model ids it can drive (worker)
    # free-form notes surfaced in `coevolve capabilities`:
    notes: str = ""
    extras: dict = field(default_factory=dict)
