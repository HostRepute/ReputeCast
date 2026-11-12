from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ImportProviderPulled")



@_attrs_define
class ImportProviderPulled:
    """ 
        Attributes:
            name (str):
            pulled (int):
            short (str):
     """

    name: str
    pulled: int
    short: str





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        pulled = self.pulled

        short = self.short


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "pulled": pulled,
            "short": short,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        pulled = d.pop("pulled")

        short = d.pop("short")

        import_provider_pulled = cls(
            name=name,
            pulled=pulled,
            short=short,
        )

        return import_provider_pulled

