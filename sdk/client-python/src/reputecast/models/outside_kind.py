from enum import StrEnum

class OutsideKind(StrEnum):
    BACKTICK = "backtick"
    CURLY_APOSTROPHE = "curly_apostrophe"
    CURLY_QUOTE = "curly_quote"
    ELLIPSIS = "ellipsis"
    EMOJI = "emoji"
    EM_DASH = "em_dash"
    EN_DASH = "en_dash"
    MINUS = "minus"
    NON_BREAKING_SPACE = "non_breaking_space"
    OTHER = "other"
    PRIME = "prime"
    THIN_SPACE = "thin_space"
    ZERO_WIDTH_SPACE = "zero_width_space"

    def __str__(self) -> str:
        return str(self.value)
