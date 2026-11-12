from enum import StrEnum

class SmsCheckKey(StrEnum):
    AUDIENCE = "audience"
    BUDGET = "budget"
    CONSENT = "consent"
    LENGTH = "length"
    LINKS = "links"
    OPT_OUT = "opt_out"
    QUIET_HOURS = "quiet_hours"
    REGISTRATION = "registration"
    ROUTES = "routes"
    SENDERS = "senders"
    SENDER_NAMED = "sender_named"
    SUPPRESSIONS = "suppressions"
    TIME_ZONES = "time_zones"

    def __str__(self) -> str:
        return str(self.value)
