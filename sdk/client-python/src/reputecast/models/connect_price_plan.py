from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectPricePlan")



@_attrs_define
class ConnectPricePlan:
    """ 
        Attributes:
            price (float):
            upto (int):
            dedicated (bool | Unset):
            free (bool | Unset):
            name (str | Unset):
            per_day (int | Unset):
            up_to (bool | Unset):
     """

    price: float
    upto: int
    dedicated: bool | Unset = UNSET
    free: bool | Unset = UNSET
    name: str | Unset = UNSET
    per_day: int | Unset = UNSET
    up_to: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        price = self.price

        upto = self.upto

        dedicated = self.dedicated

        free = self.free

        name = self.name

        per_day = self.per_day

        up_to = self.up_to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "price": price,
            "upto": upto,
        })
        if dedicated is not UNSET:
            field_dict["dedicated"] = dedicated
        if free is not UNSET:
            field_dict["free"] = free
        if name is not UNSET:
            field_dict["name"] = name
        if per_day is not UNSET:
            field_dict["perDay"] = per_day
        if up_to is not UNSET:
            field_dict["upTo"] = up_to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        price = d.pop("price")

        upto = d.pop("upto")

        dedicated = d.pop("dedicated", UNSET)

        free = d.pop("free", UNSET)

        name = d.pop("name", UNSET)

        per_day = d.pop("perDay", UNSET)

        up_to = d.pop("upTo", UNSET)

        connect_price_plan = cls(
            price=price,
            upto=upto,
            dedicated=dedicated,
            free=free,
            name=name,
            per_day=per_day,
            up_to=up_to,
        )

        return connect_price_plan

