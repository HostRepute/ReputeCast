from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReportCodeCount")



@_attrs_define
class ReportCodeCount:
    """ 
        Attributes:
            code (str): SMTP reply or provider code: 550 5.1.1
            count (int):
            provider (str): Short name: SES, Postal
            type_ (str): The provider's event name: Permanent/General, HardFail
     """

    code: str
    count: int
    provider: str
    type_: str





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        count = self.count

        provider = self.provider

        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "count": count,
            "provider": provider,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        count = d.pop("count")

        provider = d.pop("provider")

        type_ = d.pop("type")

        report_code_count = cls(
            code=code,
            count=count,
            provider=provider,
            type_=type_,
        )

        return report_code_count

