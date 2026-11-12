from enum import StrEnum

class PushWorkerStatus(StrEnum):
    BUILT_IN = "built_in"
    FOUND = "found"
    MISSING = "missing"
    NOT_A_WORKER = "not_a_worker"
    OLD_VERSION = "old_version"
    UNCHECKED = "unchecked"
    UNREACHABLE = "unreachable"

    def __str__(self) -> str:
        return str(self.value)
