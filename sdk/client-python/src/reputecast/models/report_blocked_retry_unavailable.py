from enum import StrEnum

class ReportBlockedRetryUnavailable(StrEnum):
    IS_RETRY = "is_retry"
    NONE_LEFT = "none_left"
    NOT_FINISHED = "not_finished"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
