from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ComposerTokenValue")



@_attrs_define
class ComposerTokenValue:
    """ 
        Attributes:
            fallback_used (bool):
            key (str):
            missing (bool):
            per_recipient (bool): A link filled in per recipient at send time
            token (str):  Example: {{ first_name | default: "there" }}.
            unknown (bool):
            value (str):
            audience_missing (int | Unset):
            suggestion (str | Unset):
     """

    fallback_used: bool
    key: str
    missing: bool
    per_recipient: bool
    token: str
    unknown: bool
    value: str
    audience_missing: int | Unset = UNSET
    suggestion: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        fallback_used = self.fallback_used

        key = self.key

        missing = self.missing

        per_recipient = self.per_recipient

        token = self.token

        unknown = self.unknown

        value = self.value

        audience_missing = self.audience_missing

        suggestion = self.suggestion


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fallbackUsed": fallback_used,
            "key": key,
            "missing": missing,
            "perRecipient": per_recipient,
            "token": token,
            "unknown": unknown,
            "value": value,
        })
        if audience_missing is not UNSET:
            field_dict["audienceMissing"] = audience_missing
        if suggestion is not UNSET:
            field_dict["suggestion"] = suggestion

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fallback_used = d.pop("fallbackUsed")

        key = d.pop("key")

        missing = d.pop("missing")

        per_recipient = d.pop("perRecipient")

        token = d.pop("token")

        unknown = d.pop("unknown")

        value = d.pop("value")

        audience_missing = d.pop("audienceMissing", UNSET)

        suggestion = d.pop("suggestion", UNSET)

        composer_token_value = cls(
            fallback_used=fallback_used,
            key=key,
            missing=missing,
            per_recipient=per_recipient,
            token=token,
            unknown=unknown,
            value=value,
            audience_missing=audience_missing,
            suggestion=suggestion,
        )

        return composer_token_value

