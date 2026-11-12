from enum import StrEnum

class LeadAdsImportState(StrEnum):
    DONE = "done"
    FAILED = "failed"
    RUNNING = "running"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
