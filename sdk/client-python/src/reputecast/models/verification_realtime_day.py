from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="VerificationRealtimeDay")



@_attrs_define
class VerificationRealtimeDay:
    """ 
        Attributes:
            checks (int):
            day (str): YYYY-MM-DD
     """

    checks: int
    day: str





    def to_dict(self) -> dict[str, Any]:
        checks = self.checks

        day = self.day


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checks": checks,
            "day": day,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checks = d.pop("checks")

        day = d.pop("day")

        verification_realtime_day = cls(
            checks=checks,
            day=day,
        )

        return verification_realtime_day

