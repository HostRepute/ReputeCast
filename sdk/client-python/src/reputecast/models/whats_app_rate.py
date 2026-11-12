from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_rate_category import WhatsAppRateCategory






T = TypeVar("T", bound="WhatsAppRate")



@_attrs_define
class WhatsAppRate:
    """ 
        Attributes:
            category (WhatsAppRateCategory):
            country (str):
            price (float):
     """

    category: WhatsAppRateCategory
    country: str
    price: float





    def to_dict(self) -> dict[str, Any]:
        category = self.category.value

        country = self.country

        price = self.price


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "category": category,
            "country": country,
            "price": price,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        category = WhatsAppRateCategory(d.pop("category"))




        country = d.pop("country")

        price = d.pop("price")

        whats_app_rate = cls(
            category=category,
            country=country,
            price=price,
        )

        return whats_app_rate

