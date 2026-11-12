from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_settings_input_region import VerificationSettingsInputRegion
from ..types import UNSET, Unset






T = TypeVar("T", bound="VerificationSettingsInput")



@_attrs_define
class VerificationSettingsInput:
    """ 
        Attributes:
            default_bulk (bool):
            default_forms (bool):
            price (float):
            region (VerificationSettingsInputRegion | Unset):
     """

    default_bulk: bool
    default_forms: bool
    price: float
    region: VerificationSettingsInputRegion | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        default_bulk = self.default_bulk

        default_forms = self.default_forms

        price = self.price

        region: str | Unset = UNSET
        if not isinstance(self.region, Unset):
            region = self.region.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "defaultBulk": default_bulk,
            "defaultForms": default_forms,
            "price": price,
        })
        if region is not UNSET:
            field_dict["region"] = region

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        default_bulk = d.pop("defaultBulk")

        default_forms = d.pop("defaultForms")

        price = d.pop("price")

        _region = d.pop("region", UNSET)
        region: VerificationSettingsInputRegion | Unset
        if isinstance(_region,  Unset):
            region = UNSET
        else:
            region = VerificationSettingsInputRegion(_region)




        verification_settings_input = cls(
            default_bulk=default_bulk,
            default_forms=default_forms,
            price=price,
            region=region,
        )

        return verification_settings_input

