from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
from uuid import UUID






T = TypeVar("T", bound="SmsRouteInput")



@_attrs_define
class SmsRouteInput:
    """ 
        Attributes:
            countries (list[str]):
            sender_id (UUID):
            fallback_id (str | Unset):
     """

    countries: list[str]
    sender_id: UUID
    fallback_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        countries = self.countries



        sender_id = str(self.sender_id)

        fallback_id = self.fallback_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "countries": countries,
            "senderId": sender_id,
        })
        if fallback_id is not UNSET:
            field_dict["fallbackId"] = fallback_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        countries = cast(list[str], d.pop("countries"))


        sender_id = UUID(d.pop("senderId"))




        fallback_id = d.pop("fallbackId", UNSET)

        sms_route_input = cls(
            countries=countries,
            sender_id=sender_id,
            fallback_id=fallback_id,
        )

        return sms_route_input

