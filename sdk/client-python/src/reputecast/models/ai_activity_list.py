from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.ai_activity_member import AIActivityMember
  from ..models.ai_activity_row import AIActivityRow





T = TypeVar("T", bound="AIActivityList")



@_attrs_define
class AIActivityList:
    """ 
        Attributes:
            items (list[AIActivityRow]):
            members (list[AIActivityMember]):
            month_requests (int):
            retention_days (int):
            next_ (str | Unset):
     """

    items: list[AIActivityRow]
    members: list[AIActivityMember]
    month_requests: int
    retention_days: int
    next_: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.ai_activity_member import AIActivityMember # noqa: PLC0415
        from ..models.ai_activity_row import AIActivityRow # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        members = []
        for members_item_data in self.members:
            members_item = members_item_data.to_dict()
            members.append(members_item)



        month_requests = self.month_requests

        retention_days = self.retention_days

        next_ = self.next_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "members": members,
            "monthRequests": month_requests,
            "retentionDays": retention_days,
        })
        if next_ is not UNSET:
            field_dict["next"] = next_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ai_activity_member import AIActivityMember # noqa: PLC0415
        from ..models.ai_activity_row import AIActivityRow # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = AIActivityRow.from_dict(items_item_data)



            items.append(items_item)


        members = []
        _members = d.pop("members")
        for members_item_data in (_members):
            members_item = AIActivityMember.from_dict(members_item_data)



            members.append(members_item)


        month_requests = d.pop("monthRequests")

        retention_days = d.pop("retentionDays")

        next_ = d.pop("next", UNSET)

        ai_activity_list = cls(
            items=items,
            members=members,
            month_requests=month_requests,
            retention_days=retention_days,
            next_=next_,
        )

        return ai_activity_list

