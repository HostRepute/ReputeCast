from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EarnPriceOffer")



@_attrs_define
class EarnPriceOffer:
    """ 
        Attributes:
            count (int):
            label (str):
     """

    count: int
    label: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        label = self.label


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "label": label,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        label = d.pop("label")

        earn_price_offer = cls(
            count=count,
            label=label,
        )

        return earn_price_offer

