from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AudienceTierCounts")



@_attrs_define
class AudienceTierCounts:
    """ 
        Attributes:
            active (int):
            at_risk (int):
            cooling (int):
            sunset (int):
     """

    active: int
    at_risk: int
    cooling: int
    sunset: int





    def to_dict(self) -> dict[str, Any]:
        active = self.active

        at_risk = self.at_risk

        cooling = self.cooling

        sunset = self.sunset


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "atRisk": at_risk,
            "cooling": cooling,
            "sunset": sunset,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        active = d.pop("active")

        at_risk = d.pop("atRisk")

        cooling = d.pop("cooling")

        sunset = d.pop("sunset")

        audience_tier_counts = cls(
            active=active,
            at_risk=at_risk,
            cooling=cooling,
            sunset=sunset,
        )

        return audience_tier_counts

