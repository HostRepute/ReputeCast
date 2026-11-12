from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SmsZoneNow")



@_attrs_define
class SmsZoneNow:
    """ 
        Attributes:
            inside (bool):
            local (str):
            opens (str):
            zone (str):
     """

    inside: bool
    local: str
    opens: str
    zone: str





    def to_dict(self) -> dict[str, Any]:
        inside = self.inside

        local = self.local

        opens = self.opens

        zone = self.zone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "inside": inside,
            "local": local,
            "opens": opens,
            "zone": zone,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        inside = d.pop("inside")

        local = d.pop("local")

        opens = d.pop("opens")

        zone = d.pop("zone")

        sms_zone_now = cls(
            inside=inside,
            local=local,
            opens=opens,
            zone=zone,
        )

        return sms_zone_now

