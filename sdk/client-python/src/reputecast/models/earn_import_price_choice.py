from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="EarnImportPriceChoice")



@_attrs_define
class EarnImportPriceChoice:
    """ 
        Attributes:
            legacy (bool):
            tier_id (str):
     """

    legacy: bool
    tier_id: str





    def to_dict(self) -> dict[str, Any]:
        legacy = self.legacy

        tier_id = self.tier_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "legacy": legacy,
            "tierId": tier_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        legacy = d.pop("legacy")

        tier_id = d.pop("tierId")

        earn_import_price_choice = cls(
            legacy=legacy,
            tier_id=tier_id,
        )

        return earn_import_price_choice

