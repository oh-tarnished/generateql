import threading

from pulse import LogLevel, Pulse

_pulse: Pulse | None = None
_lock = threading.Lock()


def get_pulse() -> Pulse:
    global _pulse  # noqa: PLW0603

    if _pulse is not None:
        return _pulse

    with _lock:
        if _pulse is None:
            _pulse = (
                Pulse.new()
                .with_service("loom-python-network", "1.0.0")
                .with_log_level(LogLevel.MODULE_LEVEL_2)
                .build()
                .__enter__()
            )

    return _pulse


pulse = get_pulse()
