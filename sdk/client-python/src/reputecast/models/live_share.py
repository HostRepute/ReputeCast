from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiveShare")



@_attrs_define
class LiveShare:
    """ 
        Attributes:
            key (str):
            name (str):
            percent (float):
     """

    key: str
    name: str
    percent: float





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        name = self.name

        percent = self.percent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "name": name,
            "percent": percent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        name = d.pop("name")

        percent = d.pop("percent")

        live_share = cls(
            key=key,
            name=name,
            percent=percent,
        )

        return live_share

