from enum import StrEnum

class ImportCreateInputSource(StrEnum):
    BREVO = "brevo"
    CSV = "csv"
    KEILA = "keila"
    LISTMONK = "listmonk"
    MAILCHIMP = "mailchimp"
    MAILERLITE = "mailerlite"
    NOTIFUSE = "notifuse"
    XLSX = "xlsx"

    def __str__(self) -> str:
        return str(self.value)
