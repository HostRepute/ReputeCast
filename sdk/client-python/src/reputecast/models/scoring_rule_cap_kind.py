from enum import StrEnum

class ScoringRuleCapKind(StrEnum):
    NONE = "none"
    ONCE = "once"
    ONCE_DAY = "once_day"
    ONCE_MONTH = "once_month"
    ONCE_WEEK = "once_week"
    POINTS_DAY = "points_day"
    POINTS_MONTH = "points_month"
    POINTS_WEEK = "points_week"
    WHILE_TRUE = "while_true"

    def __str__(self) -> str:
        return str(self.value)
