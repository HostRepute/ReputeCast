from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PlacementConditionalVersion")



@_attrs_define
class PlacementConditionalVersion:
    """ 
        Attributes:
            number (int):
            recipients (int):
            shows (str):
     """

    number: int
    recipients: int
    shows: str





    def to_dict(self) -> dict[str, Any]:
        number = self.number

        recipients = self.recipients

        shows = self.shows


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "number": number,
            "recipients": recipients,
            "shows": shows,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        number = d.pop("number")

        recipients = d.pop("recipients")

        shows = d.pop("shows")

        placement_conditional_version = cls(
            number=number,
            recipients=recipients,
            shows=shows,
        )

        return placement_conditional_version

