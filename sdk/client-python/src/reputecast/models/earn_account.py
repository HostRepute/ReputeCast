from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EarnAccount")



@_attrs_define
class EarnAccount:
    """ 
        Attributes:
            country (str):
            currency (str):
            id (str):
            name (str):
            tax_enabled (bool):
     """

    country: str
    currency: str
    id: str
    name: str
    tax_enabled: bool





    def to_dict(self) -> dict[str, Any]:
        country = self.country

        currency = self.currency

        id = self.id

        name = self.name

        tax_enabled = self.tax_enabled


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "country": country,
            "currency": currency,
            "id": id,
            "name": name,
            "taxEnabled": tax_enabled,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        country = d.pop("country")

        currency = d.pop("currency")

        id = d.pop("id")

        name = d.pop("name")

        tax_enabled = d.pop("taxEnabled")

        earn_account = cls(
            country=country,
            currency=currency,
            id=id,
            name=name,
            tax_enabled=tax_enabled,
        )

        return earn_account

