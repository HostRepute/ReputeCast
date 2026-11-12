from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WhatsAppFailure")



@_attrs_define
class WhatsAppFailure:
    """ 
        Attributes:
            code (str):
            count (int):
            reason (str):
     """

    code: str
    count: int
    reason: str





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        count = self.count

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "count": count,
            "reason": reason,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        count = d.pop("count")

        reason = d.pop("reason")

        whats_app_failure = cls(
            code=code,
            count=count,
            reason=reason,
        )

        return whats_app_failure

