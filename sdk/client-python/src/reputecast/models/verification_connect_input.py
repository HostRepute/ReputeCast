from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_connect_input_region import VerificationConnectInputRegion
from ..models.verification_connect_input_service import VerificationConnectInputService
from ..types import UNSET, Unset






T = TypeVar("T", bound="VerificationConnectInput")



@_attrs_define
class VerificationConnectInput:
    """ 
        Attributes:
            api_key (str):
            default_bulk (bool):
            default_forms (bool):
            price (float): Price per check in dollars
            service (VerificationConnectInputService):
            region (VerificationConnectInputRegion | Unset):
     """

    api_key: str
    default_bulk: bool
    default_forms: bool
    price: float
    service: VerificationConnectInputService
    region: VerificationConnectInputRegion | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        api_key = self.api_key

        default_bulk = self.default_bulk

        default_forms = self.default_forms

        price = self.price

        service = self.service.value

        region: str | Unset = UNSET
        if not isinstance(self.region, Unset):
            region = self.region.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "apiKey": api_key,
            "defaultBulk": default_bulk,
            "defaultForms": default_forms,
            "price": price,
            "service": service,
        })
        if region is not UNSET:
            field_dict["region"] = region

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_key = d.pop("apiKey")

        default_bulk = d.pop("defaultBulk")

        default_forms = d.pop("defaultForms")

        price = d.pop("price")

        service = VerificationConnectInputService(d.pop("service"))




        _region = d.pop("region", UNSET)
        region: VerificationConnectInputRegion | Unset
        if isinstance(_region,  Unset):
            region = UNSET
        else:
            region = VerificationConnectInputRegion(_region)




        verification_connect_input = cls(
            api_key=api_key,
            default_bulk=default_bulk,
            default_forms=default_forms,
            price=price,
            service=service,
            region=region,
        )

        return verification_connect_input

