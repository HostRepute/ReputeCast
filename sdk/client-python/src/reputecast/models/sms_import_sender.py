from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="SmsImportSender")



@_attrs_define
class SmsImportSender:
    """ 
        Attributes:
            value (str):
            countries (list[str] | Unset):
     """

    value: str
    countries: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        value = self.value

        countries: list[str] | Unset = UNSET
        if not isinstance(self.countries, Unset):
            countries = self.countries




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "value": value,
        })
        if countries is not UNSET:
            field_dict["countries"] = countries

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        value = d.pop("value")

        countries = cast(list[str], d.pop("countries", UNSET))


        sms_import_sender = cls(
            value=value,
            countries=countries,
        )

        return sms_import_sender

