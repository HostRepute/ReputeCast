from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.deliverability_incident import DeliverabilityIncident





T = TypeVar("T", bound="DeliverabilityIncidentTimeline")



@_attrs_define
class DeliverabilityIncidentTimeline:
    """ 
        Attributes:
            from_host_repute (bool):
            items (list[DeliverabilityIncident]):
            total (int):
     """

    from_host_repute: bool
    items: list[DeliverabilityIncident]
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_incident import DeliverabilityIncident # noqa: PLC0415
        from_host_repute = self.from_host_repute

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fromHostRepute": from_host_repute,
            "items": items,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_incident import DeliverabilityIncident # noqa: PLC0415
        d = dict(src_dict)
        from_host_repute = d.pop("fromHostRepute")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = DeliverabilityIncident.from_dict(items_item_data)



            items.append(items_item)


        total = d.pop("total")

        deliverability_incident_timeline = cls(
            from_host_repute=from_host_repute,
            items=items,
            total=total,
        )

        return deliverability_incident_timeline

