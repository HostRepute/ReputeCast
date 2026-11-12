from enum import StrEnum

class ConnectFinishInputSource(StrEnum):
    BREVO = "brevo"
    CSV = "csv"
    KEILA = "keila"
    LISTMONK = "listmonk"
    MAILCHIMP = "mailchimp"
    MAILERLITE = "mailerlite"

    def __str__(self) -> str:
        return str(self.value)
