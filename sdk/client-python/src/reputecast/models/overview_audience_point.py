from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="OverviewAudiencePoint")



@_attrs_define
class OverviewAudiencePoint:
    """ 
        Attributes:
            day (datetime.date):
            subscribers (int):
     """

    day: datetime.date
    subscribers: int





    def to_dict(self) -> dict[str, Any]:
        day = self.day.isoformat()

        subscribers = self.subscribers


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "day": day,
            "subscribers": subscribers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        day = datetime.date.fromisoformat(d.pop("day"))




        subscribers = d.pop("subscribers")

        overview_audience_point = cls(
            day=day,
            subscribers=subscribers,
        )

        return overview_audience_point

