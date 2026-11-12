from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsFailureCode")



@_attrs_define
class SmsFailureCode:
    """ 
        Attributes:
            code (str):
            count (int):
            provider (str):
     """

    code: str
    count: int
    provider: str





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        count = self.count

        provider = self.provider


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "count": count,
            "provider": provider,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        count = d.pop("count")

        provider = d.pop("provider")

        sms_failure_code = cls(
            code=code,
            count=count,
            provider=provider,
        )

        return sms_failure_code

