from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GiftTier")



@_attrs_define
class GiftTier:
    """ 
        Attributes:
            id (str):
            monthly_cents (int):
            name (str):
            short (str):
            yearly_cents (int):
     """

    id: str
    monthly_cents: int
    name: str
    short: str
    yearly_cents: int





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        monthly_cents = self.monthly_cents

        name = self.name

        short = self.short

        yearly_cents = self.yearly_cents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "monthlyCents": monthly_cents,
            "name": name,
            "short": short,
            "yearlyCents": yearly_cents,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        monthly_cents = d.pop("monthlyCents")

        name = d.pop("name")

        short = d.pop("short")

        yearly_cents = d.pop("yearlyCents")

        gift_tier = cls(
            id=id,
            monthly_cents=monthly_cents,
            name=name,
            short=short,
            yearly_cents=yearly_cents,
        )

        return gift_tier

