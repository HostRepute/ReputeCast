from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_status_count_status import ProviderStatusCountStatus






T = TypeVar("T", bound="ProviderStatusCount")



@_attrs_define
class ProviderStatusCount:
    """ 
        Attributes:
            count (int):
            status (ProviderStatusCountStatus):
     """

    count: int
    status: ProviderStatusCountStatus





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        status = ProviderStatusCountStatus(d.pop("status"))




        provider_status_count = cls(
            count=count,
            status=status,
        )

        return provider_status_count

