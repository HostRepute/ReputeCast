from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.automation_alert import AutomationAlert
  from ..models.automation_list_item import AutomationListItem
  from ..models.automation_route import AutomationRoute
  from ..models.automation_status_counts import AutomationStatusCounts





T = TypeVar("T", bound="AutomationList")



@_attrs_define
class AutomationList:
    """ 
        Attributes:
            alerts (list[AutomationAlert]):
            counts (AutomationStatusCounts):
            days (int):
            items (list[AutomationListItem]):
            route (AutomationRoute):
     """

    alerts: list[AutomationAlert]
    counts: AutomationStatusCounts
    days: int
    items: list[AutomationListItem]
    route: AutomationRoute





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_alert import AutomationAlert # noqa: PLC0415
        from ..models.automation_list_item import AutomationListItem # noqa: PLC0415
        from ..models.automation_route import AutomationRoute # noqa: PLC0415
        from ..models.automation_status_counts import AutomationStatusCounts # noqa: PLC0415
        alerts = []
        for alerts_item_data in self.alerts:
            alerts_item = alerts_item_data.to_dict()
            alerts.append(alerts_item)



        counts = self.counts.to_dict()

        days = self.days

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        route = self.route.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alerts": alerts,
            "counts": counts,
            "days": days,
            "items": items,
            "route": route,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_alert import AutomationAlert # noqa: PLC0415
        from ..models.automation_list_item import AutomationListItem # noqa: PLC0415
        from ..models.automation_route import AutomationRoute # noqa: PLC0415
        from ..models.automation_status_counts import AutomationStatusCounts # noqa: PLC0415
        d = dict(src_dict)
        alerts = []
        _alerts = d.pop("alerts")
        for alerts_item_data in (_alerts):
            alerts_item = AutomationAlert.from_dict(alerts_item_data)



            alerts.append(alerts_item)


        counts = AutomationStatusCounts.from_dict(d.pop("counts"))




        days = d.pop("days")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = AutomationListItem.from_dict(items_item_data)



            items.append(items_item)


        route = AutomationRoute.from_dict(d.pop("route"))




        automation_list = cls(
            alerts=alerts,
            counts=counts,
            days=days,
            items=items,
            route=route,
        )

        return automation_list

