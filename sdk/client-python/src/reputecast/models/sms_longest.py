from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsLongest")



@_attrs_define
class SmsLongest:
    """ 
        Attributes:
            characters (int):
            cost (float):
            name (str):
            recipients (int):
            segments (int):
            value (str):
     """

    characters: int
    cost: float
    name: str
    recipients: int
    segments: int
    value: str





    def to_dict(self) -> dict[str, Any]:
        characters = self.characters

        cost = self.cost

        name = self.name

        recipients = self.recipients

        segments = self.segments

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "characters": characters,
            "cost": cost,
            "name": name,
            "recipients": recipients,
            "segments": segments,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        characters = d.pop("characters")

        cost = d.pop("cost")

        name = d.pop("name")

        recipients = d.pop("recipients")

        segments = d.pop("segments")

        value = d.pop("value")

        sms_longest = cls(
            characters=characters,
            cost=cost,
            name=name,
            recipients=recipients,
            segments=segments,
            value=value,
        )

        return sms_longest

