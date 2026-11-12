from enum import StrEnum

class ImportLogEntryKind(StrEnum):
    DONE = "done"
    ERROR = "error"
    INFO = "info"
    RETRY = "retry"
    RUNNING = "running"
    START = "start"
    SYNC = "sync"

    def __str__(self) -> str:
        return str(self.value)
