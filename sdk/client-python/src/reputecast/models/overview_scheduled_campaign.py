from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.overview_readiness import OverviewReadiness
  from ..models.overview_route_share import OverviewRouteShare





T = TypeVar("T", bound="OverviewScheduledCampaign")



@_attrs_define
class OverviewScheduledCampaign:
    """ 
        Attributes:
            cost (float):
            id (str):
            name (str):
            recipients (int):
            routes (list[OverviewRouteShare]):
            scheduled_at (datetime.datetime):
            timezone (str):
            duration_minutes (float | Unset):
            readiness (OverviewReadiness | Unset):
     """

    cost: float
    id: str
    name: str
    recipients: int
    routes: list[OverviewRouteShare]
    scheduled_at: datetime.datetime
    timezone: str
    duration_minutes: float | Unset = UNSET
    readiness: OverviewReadiness | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.overview_readiness import OverviewReadiness # noqa: PLC0415
        from ..models.overview_route_share import OverviewRouteShare # noqa: PLC0415
        cost = self.cost

        id = self.id

        name = self.name

        recipients = self.recipients

        routes = []
        for routes_item_data in self.routes:
            routes_item = routes_item_data.to_dict()
            routes.append(routes_item)



        scheduled_at = self.scheduled_at.isoformat()

        timezone = self.timezone

        duration_minutes = self.duration_minutes

        readiness: dict[str, Any] | Unset = UNSET
        if not isinstance(self.readiness, Unset):
            readiness = self.readiness.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cost": cost,
            "id": id,
            "name": name,
            "recipients": recipients,
            "routes": routes,
            "scheduledAt": scheduled_at,
            "timezone": timezone,
        })
        if duration_minutes is not UNSET:
            field_dict["durationMinutes"] = duration_minutes
        if readiness is not UNSET:
            field_dict["readiness"] = readiness

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.overview_readiness import OverviewReadiness # noqa: PLC0415
        from ..models.overview_route_share import OverviewRouteShare # noqa: PLC0415
        d = dict(src_dict)
        cost = d.pop("cost")

        id = d.pop("id")

        name = d.pop("name")

        recipients = d.pop("recipients")

        routes = []
        _routes = d.pop("routes")
        for routes_item_data in (_routes):
            routes_item = OverviewRouteShare.from_dict(routes_item_data)



            routes.append(routes_item)


        scheduled_at = datetime.datetime.fromisoformat(d.pop("scheduledAt"))




        timezone = d.pop("timezone")

        duration_minutes = d.pop("durationMinutes", UNSET)

        _readiness = d.pop("readiness", UNSET)
        readiness: OverviewReadiness | Unset
        if isinstance(_readiness,  Unset):
            readiness = UNSET
        else:
            readiness = OverviewReadiness.from_dict(_readiness)




        overview_scheduled_campaign = cls(
            cost=cost,
            id=id,
            name=name,
            recipients=recipients,
            routes=routes,
            scheduled_at=scheduled_at,
            timezone=timezone,
            duration_minutes=duration_minutes,
            readiness=readiness,
        )

        return overview_scheduled_campaign

