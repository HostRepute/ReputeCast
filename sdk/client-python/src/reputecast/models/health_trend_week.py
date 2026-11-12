from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="HealthTrendWeek")



@_attrs_define
class HealthTrendWeek:
    """ 
        Attributes:
            active (float):
            at_risk (float):
            cooling (float):
            day (datetime.date):
            sunset (float):
     """

    active: float
    at_risk: float
    cooling: float
    day: datetime.date
    sunset: float





    def to_dict(self) -> dict[str, Any]:
        active = self.active

        at_risk = self.at_risk

        cooling = self.cooling

        day = self.day.isoformat()

        sunset = self.sunset


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "atRisk": at_risk,
            "cooling": cooling,
            "day": day,
            "sunset": sunset,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        active = d.pop("active")

        at_risk = d.pop("atRisk")

        cooling = d.pop("cooling")

        day = datetime.date.fromisoformat(d.pop("day"))




        sunset = d.pop("sunset")

        health_trend_week = cls(
            active=active,
            at_risk=at_risk,
            cooling=cooling,
            day=day,
            sunset=sunset,
        )

        return health_trend_week

