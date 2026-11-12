from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WhatsAppCostCountry")



@_attrs_define
class WhatsAppCostCountry:
    """ 
        Attributes:
            amount (float):
            country (str):
            messages (int):
            price (float):
     """

    amount: float
    country: str
    messages: int
    price: float





    def to_dict(self) -> dict[str, Any]:
        amount = self.amount

        country = self.country

        messages = self.messages

        price = self.price


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amount": amount,
            "country": country,
            "messages": messages,
            "price": price,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount = d.pop("amount")

        country = d.pop("country")

        messages = d.pop("messages")

        price = d.pop("price")

        whats_app_cost_country = cls(
            amount=amount,
            country=country,
            messages=messages,
            price=price,
        )

        return whats_app_cost_country

