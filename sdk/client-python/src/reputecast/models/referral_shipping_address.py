from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralShippingAddress")



@_attrs_define
class ReferralShippingAddress:
    """ 
        Attributes:
            city (str):
            country (str):
            name (str):
            street (str):
     """

    city: str
    country: str
    name: str
    street: str





    def to_dict(self) -> dict[str, Any]:
        city = self.city

        country = self.country

        name = self.name

        street = self.street


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "city": city,
            "country": country,
            "name": name,
            "street": street,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        city = d.pop("city")

        country = d.pop("country")

        name = d.pop("name")

        street = d.pop("street")

        referral_shipping_address = cls(
            city=city,
            country=country,
            name=name,
            street=street,
        )

        return referral_shipping_address

