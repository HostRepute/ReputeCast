from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="LiveDetails")



@_attrs_define
class LiveDetails:
    """ 
        Attributes:
            audience (str):
            audience_count (int):
            cost (float):
            cost_planned (float):
            from_ (str):
            route (str):
            scheduled_by (str):
            watch_hours (int):
     """

    audience: str
    audience_count: int
    cost: float
    cost_planned: float
    from_: str
    route: str
    scheduled_by: str
    watch_hours: int





    def to_dict(self) -> dict[str, Any]:
        audience = self.audience

        audience_count = self.audience_count

        cost = self.cost

        cost_planned = self.cost_planned

        from_ = self.from_

        route = self.route

        scheduled_by = self.scheduled_by

        watch_hours = self.watch_hours


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "audience": audience,
            "audienceCount": audience_count,
            "cost": cost,
            "costPlanned": cost_planned,
            "from": from_,
            "route": route,
            "scheduledBy": scheduled_by,
            "watchHours": watch_hours,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        audience = d.pop("audience")

        audience_count = d.pop("audienceCount")

        cost = d.pop("cost")

        cost_planned = d.pop("costPlanned")

        from_ = d.pop("from")

        route = d.pop("route")

        scheduled_by = d.pop("scheduledBy")

        watch_hours = d.pop("watchHours")

        live_details = cls(
            audience=audience,
            audience_count=audience_count,
            cost=cost,
            cost_planned=cost_planned,
            from_=from_,
            route=route,
            scheduled_by=scheduled_by,
            watch_hours=watch_hours,
        )

        return live_details

