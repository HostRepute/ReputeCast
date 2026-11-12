from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReportProviderCount")



@_attrs_define
class ReportProviderCount:
    """ 
        Attributes:
            count (int):
            provider_id (str):
            short (str):
     """

    count: int
    provider_id: str
    short: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        provider_id = self.provider_id

        short = self.short


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "providerId": provider_id,
            "short": short,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        report_provider_count = cls(
            count=count,
            provider_id=provider_id,
            short=short,
        )

        return report_provider_count

