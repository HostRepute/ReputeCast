from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppWindowExample")



@_attrs_define
class WhatsAppWindowExample:
    """ 
        Attributes:
            name (str):
            until (datetime.datetime):
     """

    name: str
    until: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "until": until,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        until = datetime.datetime.fromisoformat(d.pop("until"))




        whats_app_window_example = cls(
            name=name,
            until=until,
        )

        return whats_app_window_example

