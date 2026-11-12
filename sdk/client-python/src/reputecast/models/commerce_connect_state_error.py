from enum import StrEnum

class CommerceConnectStateError(StrEnum):
    NOT_INSTALLED = "not_installed"
    SECRET_REFUSED = "secret_refused"
    STATE_EXPIRED = "state_expired"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
