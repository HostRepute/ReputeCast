from enum import StrEnum

class ReportComparisonMetric(StrEnum):
    BOUNCES = "bounces"
    CLICKS = "clicks"
    COMPLAINTS = "complaints"
    DELIVERED = "delivered"
    HUMANOPENS = "humanOpens"
    PRIVACYOPENS = "privacyOpens"
    SENT = "sent"
    UNSUBSCRIBES = "unsubscribes"

    def __str__(self) -> str:
        return str(self.value)
