from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiveABVariant")



@_attrs_define
class LiveABVariant:
    """ 
        Attributes:
            key (str):
            rate (float):
            sent (int):
     """

    key: str
    rate: float
    sent: int





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        rate = self.rate

        sent = self.sent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "rate": rate,
            "sent": sent,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        rate = d.pop("rate")

        sent = d.pop("sent")

        live_ab_variant = cls(
            key=key,
            rate=rate,
            sent=sent,
        )

        return live_ab_variant

