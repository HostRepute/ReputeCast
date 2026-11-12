from enum import StrEnum

class SmsPauseViewMetric(StrEnum):
    FAILED = "failed"
    FILTERED = "filtered"
    OPT_OUTS = "opt_outs"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
