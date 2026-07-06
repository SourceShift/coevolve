"""Integration registry — the extension point.

Two ways to add an integration, neither touching core:
  1. In-tree: decorate the class with ``@register`` (built-ins do this).
  2. Third-party: expose a ``coevolve.integrations`` entry point that imports a
     module which registers on import. ``discover()`` loads them.

Core resolves backends by role (``Kind``) + name, and by capability, never by
concrete import.
"""
from __future__ import annotations

from importlib import metadata
from typing import TypeVar

from .capabilities import Kind
from .integrations.base import Integration

_CLASSES: dict[Kind, dict[str, type[Integration]]] = {}
_INSTANCES: dict[tuple[Kind, str], Integration] = {}
_discovered = False

T = TypeVar("T", bound=Integration)

ENTRY_POINT_GROUP = "coevolve.integrations"


def register(cls: type[T]) -> type[T]:
    """Class decorator: register an Integration under its capabilities().kind."""
    if not getattr(cls, "name", ""):
        raise ValueError(f"{cls.__name__} must set a class-level `name`")
    kind = cls().capabilities().kind  # cheap: capabilities() must be pure
    _CLASSES.setdefault(kind, {})[cls.name] = cls
    return cls


def discover() -> None:
    """Load third-party integrations advertised via entry points (idempotent)."""
    global _discovered
    if _discovered:
        return
    _discovered = True
    try:
        eps = metadata.entry_points(group=ENTRY_POINT_GROUP)
    except TypeError:  # <3.10 API shape
        eps = metadata.entry_points().get(ENTRY_POINT_GROUP, [])  # type: ignore[attr-defined]
    for ep in eps:
        try:
            ep.load()  # module self-registers on import
        except Exception:  # a broken plugin must not break core
            continue


def get(kind: Kind, name: str) -> Integration:
    discover()
    inst = _INSTANCES.get((kind, name))
    if inst is None:
        cls = _CLASSES.get(kind, {}).get(name)
        if cls is None:
            raise KeyError(f"no {kind.value} integration named {name!r} "
                           f"(have: {sorted(_CLASSES.get(kind, {}))})")
        inst = _INSTANCES[(kind, name)] = cls()
    return inst


def all_of(kind: Kind) -> list[Integration]:
    discover()
    return [get(kind, n) for n in sorted(_CLASSES.get(kind, {}))]


def first_of(kind: Kind) -> Integration | None:
    items = all_of(kind)
    return items[0] if items else None


def names(kind: Kind) -> list[str]:
    discover()
    return sorted(_CLASSES.get(kind, {}))


def registered() -> dict[str, list[str]]:
    """Snapshot for `coevolve capabilities`."""
    discover()
    return {k.value: sorted(v) for k, v in _CLASSES.items()}
