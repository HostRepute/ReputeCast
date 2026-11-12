from enum import StrEnum

class ConnectSMTPFormProtocol(StrEnum):
    IMAP = "imap"
    POP3 = "pop3"

    def __str__(self) -> str:
        return str(self.value)
