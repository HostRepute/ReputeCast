from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="SmsSenderUpdate")



@_attrs_define
class SmsSenderUpdate:
    """ 
        Attributes:
            countries (list[str]):
     """

    countries: list[str]





    def to_dict(self) -> dict[str, Any]:
        countries = self.countries




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "countries": countries,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        countries = cast(list[str], d.pop("countries"))


        sms_sender_update = cls(
            countries=countries,
        )

        return sms_sender_update

