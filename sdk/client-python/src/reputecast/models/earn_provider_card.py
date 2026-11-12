from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_provider_card_key import EarnProviderCardKey
from typing import cast






T = TypeVar("T", bound="EarnProviderCard")



@_attrs_define
class EarnProviderCard:
    """ 
        Attributes:
            available (bool):
            capabilities (list[str]):
            key (EarnProviderCardKey):
            merchant_of_record (bool):
            name (str):
     """

    available: bool
    capabilities: list[str]
    key: EarnProviderCardKey
    merchant_of_record: bool
    name: str





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        capabilities = self.capabilities



        key = self.key.value

        merchant_of_record = self.merchant_of_record

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "capabilities": capabilities,
            "key": key,
            "merchantOfRecord": merchant_of_record,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        capabilities = cast(list[str], d.pop("capabilities"))


        key = EarnProviderCardKey(d.pop("key"))




        merchant_of_record = d.pop("merchantOfRecord")

        name = d.pop("name")

        earn_provider_card = cls(
            available=available,
            capabilities=capabilities,
            key=key,
            merchant_of_record=merchant_of_record,
            name=name,
        )

        return earn_provider_card

