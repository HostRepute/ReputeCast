from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="IncidentRate")



@_attrs_define
class IncidentRate:
    """ 
        Attributes:
            base (int):
            events (int):
            limit (float):
            metric (str):
            value (float):
     """

    base: int
    events: int
    limit: float
    metric: str
    value: float





    def to_dict(self) -> dict[str, Any]:
        base = self.base

        events = self.events

        limit = self.limit

        metric = self.metric

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "base": base,
            "events": events,
            "limit": limit,
            "metric": metric,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base = d.pop("base")

        events = d.pop("events")

        limit = d.pop("limit")

        metric = d.pop("metric")

        value = d.pop("value")

        incident_rate = cls(
            base=base,
            events=events,
            limit=limit,
            metric=metric,
            value=value,
        )

        return incident_rate

