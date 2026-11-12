from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.audit_actor_ref import AuditActorRef
  from ..models.audit_category import AuditCategory
  from ..models.audit_event_item import AuditEventItem
  from ..models.audit_stream_status import AuditStreamStatus





T = TypeVar("T", bound="AuditEventList")



@_attrs_define
class AuditEventList:
    """ 
        Attributes:
            action_groups (list[str]):
            actors (list[AuditActorRef]):
            categories (list[AuditCategory]):
            items (list[AuditEventItem]):
            retention_days (int):
            stream (AuditStreamStatus):
            total (int):
            next_cursor (str | Unset):
     """

    action_groups: list[str]
    actors: list[AuditActorRef]
    categories: list[AuditCategory]
    items: list[AuditEventItem]
    retention_days: int
    stream: AuditStreamStatus
    total: int
    next_cursor: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.audit_actor_ref import AuditActorRef # noqa: PLC0415
        from ..models.audit_category import AuditCategory # noqa: PLC0415
        from ..models.audit_event_item import AuditEventItem # noqa: PLC0415
        from ..models.audit_stream_status import AuditStreamStatus # noqa: PLC0415
        action_groups = self.action_groups



        actors = []
        for actors_item_data in self.actors:
            actors_item = actors_item_data.to_dict()
            actors.append(actors_item)



        categories = []
        for categories_item_data in self.categories:
            categories_item = categories_item_data.to_dict()
            categories.append(categories_item)



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        retention_days = self.retention_days

        stream = self.stream.to_dict()

        total = self.total

        next_cursor = self.next_cursor


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "actionGroups": action_groups,
            "actors": actors,
            "categories": categories,
            "items": items,
            "retentionDays": retention_days,
            "stream": stream,
            "total": total,
        })
        if next_cursor is not UNSET:
            field_dict["nextCursor"] = next_cursor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.audit_actor_ref import AuditActorRef # noqa: PLC0415
        from ..models.audit_category import AuditCategory # noqa: PLC0415
        from ..models.audit_event_item import AuditEventItem # noqa: PLC0415
        from ..models.audit_stream_status import AuditStreamStatus # noqa: PLC0415
        d = dict(src_dict)
        action_groups = cast(list[str], d.pop("actionGroups"))


        actors = []
        _actors = d.pop("actors")
        for actors_item_data in (_actors):
            actors_item = AuditActorRef.from_dict(actors_item_data)



            actors.append(actors_item)


        categories = []
        _categories = d.pop("categories")
        for categories_item_data in (_categories):
            categories_item = AuditCategory.from_dict(categories_item_data)



            categories.append(categories_item)


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = AuditEventItem.from_dict(items_item_data)



            items.append(items_item)


        retention_days = d.pop("retentionDays")

        stream = AuditStreamStatus.from_dict(d.pop("stream"))




        total = d.pop("total")

        next_cursor = d.pop("nextCursor", UNSET)

        audit_event_list = cls(
            action_groups=action_groups,
            actors=actors,
            categories=categories,
            items=items,
            retention_days=retention_days,
            stream=stream,
            total=total,
            next_cursor=next_cursor,
        )

        return audit_event_list

