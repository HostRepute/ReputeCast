from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="VerificationServiceChoice")



@_attrs_define
class VerificationServiceChoice:
    """ 
        Attributes:
            default (bool):
            key (str):
            name (str):
            price (float):
            region_label (str):
            balance (int | Unset):
     """

    default: bool
    key: str
    name: str
    price: float
    region_label: str
    balance: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        default = self.default

        key = self.key

        name = self.name

        price = self.price

        region_label = self.region_label

        balance = self.balance


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "default": default,
            "key": key,
            "name": name,
            "price": price,
            "regionLabel": region_label,
        })
        if balance is not UNSET:
            field_dict["balance"] = balance

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        default = d.pop("default")

        key = d.pop("key")

        name = d.pop("name")

        price = d.pop("price")

        region_label = d.pop("regionLabel")

        balance = d.pop("balance", UNSET)

        verification_service_choice = cls(
            default=default,
            key=key,
            name=name,
            price=price,
            region_label=region_label,
            balance=balance,
        )

        return verification_service_choice

