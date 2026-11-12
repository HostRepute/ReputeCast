from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
from uuid import UUID






T = TypeVar("T", bound="SmsSenderIDInput")



@_attrs_define
class SmsSenderIDInput:
    """ 
        Attributes:
            countries (list[str]):
            provider_id (UUID):
            value (str):
     """

    countries: list[str]
    provider_id: UUID
    value: str





    def to_dict(self) -> dict[str, Any]:
        countries = self.countries



        provider_id = str(self.provider_id)

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "countries": countries,
            "providerId": provider_id,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        countries = cast(list[str], d.pop("countries"))


        provider_id = UUID(d.pop("providerId"))




        value = d.pop("value")

        sms_sender_id_input = cls(
            countries=countries,
            provider_id=provider_id,
            value=value,
        )

        return sms_sender_id_input

